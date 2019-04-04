package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"unicode/utf8"
)

var ErrInvalidText = errors.New("unavailable encoding")
var ErrInternal = errors.New("internal error")

type Line struct {
	Num uint
	Str string
}

// cap(q) is required always greater than 1
type LineQueue struct{ q chan *Line }

func NewLineQueue(capacity uint) (*LineQueue, error) {
	if capacity == 0 {
		return nil, errors.New("capacity is 0")
	}
	return &LineQueue{make(chan *Line, capacity)}, nil
}

func (lq *LineQueue) Len() int { return len(lq.q) }
func (lq *LineQueue) Cap() int { return cap(lq.q) }

func (lq *LineQueue) Push(l *Line) {
	select {
	case lq.q <- l:
	default:
		<-lq.q
		lq.q <- l
	}
}

func (lq *LineQueue) PopAll() []*Line {
	lines := make([]*Line, 0, cap(lq.q))
	for len(lq.q) != 0 {
		lines = append(lines, <-lq.q)
	}
	return lines
}

// change to struct{ raw []*Line, index int, pos int }?
type Context struct {
	line   *Line
	before []*Line
	after  []*Line
}

func FprintContexts(writer io.Writer, prefix string, cs []*Context) error {
	if cs == nil {
		return nil
	}
	var err error
	f := func(ls []*Line) {
		for _, l := range ls {
			_, err = fmt.Fprintf(writer, "%s%d-%s\n", prefix, l.Num, l.Str)
		}
	}
	for _, c := range cs {
		f(c.before)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(writer, "%s%d:%s\n", prefix, c.line.Num, c.line.Str)
		if err != nil {
			return err
		}
		f(c.after)
		if err != nil {
			return err
		}
	}
	return nil
}

type File struct {
	path string
	cs   []*Context
}

func FprintFile(writer io.Writer, f *File) error {
	var err error
	if len(f.cs) == 0 {
		return nil
	}
	_, err = fmt.Fprintln(writer, f.path)
	if err != nil {
		return err
	}
	err = FprintContexts(writer, "", f.cs)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer)
	if err != nil {
		return err
	}
	return nil
}

func FprintFileVerbose(writer io.Writer, f *File) error {
	var err error
	if len(f.cs) == 0 {
		return nil
	}
	err = FprintContexts(writer, f.path, f.cs)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer)
	if err != nil {
		return err
	}
	return nil
}

type Walker struct {
	fileQueue chan string
	dirQueue  chan []string

	regexp *regexp.Regexp
	files  map[string]*File

	log, errlog   *log.Logger
	nworker       int
	mu            sync.Mutex
	wg            sync.WaitGroup
	once          sync.Once
	internalError bool
}

func NewWalker() *Walker {
	nworker := runtime.NumCPU() / 2
	if nworker < 2 {
		nworker = 2
	}
	return &Walker{
		fileQueue: make(chan string, 256),
		dirQueue:  make(chan []string, 128),
		nworker:   nworker,
		files:     make(map[string]*File),
		log:       log.New(ioutil.Discard, Name+":", 0),
		errlog:    log.New(ioutil.Discard, Name+":[Err]:", 0),
	}
}

func (w *Walker) SetLog(writer io.Writer)    { w.log.SetOutput(writer) }
func (w *Walker) SetErrLog(writer io.Writer) { w.errlog.SetOutput(writer) }

func (w *Walker) setInternalError() {
	w.once.Do(func() { w.internalError = true })
}

// may change verbose
func (w *Walker) run(writer io.Writer, verbose bool, pat string, lines uint, paths ...string) error {
	re, err := regexp.Compile(pat)
	if err != nil {
		return err
	}
	w.regexp = re

	go w.dirWalker()
	var doResult func(*File)
	if writer == nil {
		doResult = func(f *File) {
			w.mu.Lock()
			w.files[f.path] = f
			w.mu.Unlock()
		}
	} else {
		var printFunc func(io.Writer, *File) error
		if verbose {
			printFunc = FprintFileVerbose
		} else {
			printFunc = FprintFile
		}
		var rwm sync.RWMutex
		doResult = func(f *File) {
			rwm.Lock()
			err := printFunc(writer, f)
			rwm.Unlock()
			if err != nil {
				panic(err)
			}
		}
	}
	for i := 0; i != w.nworker; i++ {
		go w.fileWalker(lines, doResult)
	}

	// paths to abs?
	// treat symlinks?
	// limitation of depth?
	var dirs []string
	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		switch {
		case fi.Mode().IsRegular():
			w.wg.Add(1)
			w.fileQueue <- path
		case fi.IsDir():
			dirs = append(dirs, path)
		}
	}
	w.wg.Add(1)
	w.dirQueue <- dirs
	w.wg.Wait()

	if w.internalError {
		return ErrInternal
	}
	return nil
}

func (w *Walker) Run(pat string, lines uint, paths ...string) error {
	return w.run(nil, false, pat, lines, paths...)
}

func (w *Walker) RunSyncWrite(writer io.Writer, verbose bool, pat string, lines uint, paths ...string) error {
	return w.run(writer, verbose, pat, lines, paths...)
}

// for goroutine
// send tasks to {file,dir}Queue
func (w *Walker) dirWalker() {
	var nextDirs []string
	for ; true; w.wg.Done() {
		dirs := <-w.dirQueue
		for _, dir := range dirs {
			fis, err := ioutil.ReadDir(dir)
			if err != nil {
				w.setInternalError()
				w.errlog.Printf("%q:%v", dir, err)
				if os.IsNotExist(err) || os.IsPermission(err) {
					continue
				}
				// unexpected error
				panic(err)
			}
			for _, fi := range fis {
				switch {
				case fi.Mode().IsRegular():
					w.wg.Add(1)
					w.fileQueue <- filepath.Join(dir, fi.Name())
				case fi.IsDir():
					nextDirs = append(nextDirs, filepath.Join(dir, fi.Name()))
				}
			}
		}
		if nextDirs != nil {
			w.wg.Add(1)
			w.dirQueue <- nextDirs
			nextDirs = nil
		}
	}
}

// for goroutine
func (w *Walker) fileWalker(lines uint, doResult func(*File)) {
	var lq *LineQueue
	if lines != 0 {
		// not need error check
		lq, _ = NewLineQueue(lines)
	}
	for ; true; w.wg.Done() {
		file := <-w.fileQueue
		w.mu.Lock()
		if _, ok := w.files[file]; ok {
			w.mu.Unlock()
			continue
		}
		w.files[file] = nil
		w.mu.Unlock()
		cs, err := w.readFile(file, lq)
		if err != nil {
			w.setInternalError()
			w.errlog.Printf("%q:%v", file, err)
			if os.IsNotExist(err) || os.IsPermission(err) {
				continue
			}
			if err == bufio.ErrTooLong {
				continue
			}
			if err == ErrInvalidText {
				continue
			}
			// unexpected error
			panic(err)
		}
		w.log.Println(file)
		if len(cs) != 0 {
			doResult(&File{
				path: file,
				cs:   cs,
			})
		}
	}
}

func (w *Walker) readFile(file string, lq *LineQueue) ([]*Context, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cs []*Context
	var c = new(Context)
	var txt string
	var i uint
	var matched bool

	var csAdd func()
	if lq == nil {
		csAdd = func() {
			if matched {
				cs = append(cs, &Context{
					line:   &Line{i, txt},
					before: []*Line{},
					after:  []*Line{},
				})
			}
		}
	} else {
		csAdd = func() {
			if c.line != nil {
				if matched {
					c.after = lq.PopAll()
					cs = append(cs, c)
					c = &Context{
						before: []*Line{},
						line:   &Line{i, txt},
						after:  []*Line{},
					}
					return
				}
				if lq.Len() == lq.Cap() {
					c.after = lq.PopAll()
					cs = append(cs, c)
					c = new(Context)
				}
			} else if matched {
				c.before = lq.PopAll()
				c.line = &Line{i, txt}
				return
			}
			lq.Push(&Line{i, txt})
		}
	}

	sc := bufio.NewScanner(f)
	for i = uint(1); sc.Scan(); i++ {
		if i == 0 {
			return nil, errors.New("too many lines")
		}
		txt = sc.Text()
		if !utf8.ValidString(txt) {
			return nil, ErrInvalidText
		}
		matched = w.regexp.MatchString(txt)
		csAdd()
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}

	// append last one for w.lines != 0
	if c.line != nil {
		c.after = lq.PopAll()
		cs = append(cs, c)
	}
	return cs, nil
}

func (w *Walker) fprintFiles(writer io.Writer, verbose bool) error {
	var err error
	printFunc := FprintFile
	if verbose {
		printFunc = FprintFileVerbose
	}
	for _, f := range w.files {
		if f == nil {
			continue
		}
		err = printFunc(writer, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Walker) FprintFiles(writer io.Writer) error {
	return w.fprintFiles(writer, false)
}

func (w *Walker) FprintFilesVerbose(writer io.Writer) error {
	return w.fprintFiles(writer, true)
}
