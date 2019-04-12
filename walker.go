package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

var ErrAlreadyStarted = errors.New("Walker: already started")

type Walker struct {
	fileQueue chan string
	dirQueue  chan []string

	// store checked files path.
	checked map[string]bool

	// for fileWalker.
	re      *regexp.Regexp
	nbefore int
	nafter  int

	mu sync.Mutex
	wg sync.WaitGroup

	// errorhandler is for dirWalker and fileWalker.
	// if unexpected error coming then to panic is better.
	errorHandler func(error)

	isStarted bool
	exitcode  int
}

func NewWalker() *Walker {
	return &Walker{
		checked:      make(map[string]bool),
		errorHandler: DefaultErrorHandler,
	}
}

var DefaultErrorHandler = func(err error) {
	if os.IsNotExist(err) || os.IsPermission(err) {
		return
	}
	if _, ok := err.(*ExpectedError); ok {
		return
	}
	// unexpeted error
	panic(err)
}

func (w *Walker) SetErrorHandler(f func(error)) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isStarted {
		return ErrAlreadyStarted
	}
	w.errorHandler = f
	return nil
}

func (w *Walker) SetRegexp(pat string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isStarted {
		return ErrAlreadyStarted
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return err
	}
	w.re = re
	return nil
}

func (w *Walker) SetContext(nbefore, nafter int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isStarted {
		return ErrAlreadyStarted
	}
	w.nbefore = nbefore
	w.nafter = nafter
	return nil
}

func (w *Walker) SendPath(paths ...string) error {
	var dirs []string
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			dirs = append(dirs, abs)
		} else if fi.Mode().IsRegular() {
			w.wg.Add(1)
			w.fileQueue <- abs
		}
	}
	if len(dirs) != 0 {
		w.wg.Add(1)
		w.dirQueue <- dirs
	}
	return nil
}

func (w *Walker) Start() (resultReceiver <-chan *File, wait func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	nworker := runtime.NumCPU() / 4
	if nworker < 2 {
		nworker = 2
	}
	nfileQueue := 128

	done := make(chan struct{})
	rq := make(chan *File, nfileQueue)

	errQueue := make(chan error, nfileQueue)
	go w.handleError(errQueue, w.errorHandler)

	w.dirQueue = make(chan []string, nworker)
	w.fileQueue = make(chan string, nfileQueue)
	for i := 0; i != nworker; i++ {
		go w.dirWalker(done, errQueue)
		go w.fileWalker(done, rq, errQueue)
	}

	w.isStarted = true
	return rq, func() {
		w.wg.Wait()
		close(errQueue)
		close(done)
		close(rq)
		w.mu.Lock()
		w.isStarted = false
		w.mu.Unlock()
	}
}

func (w *Walker) WaitExitCode() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isStarted {
		w.wg.Wait()
		return w.exitcode
	}
	return w.exitcode
}

func (w *Walker) handleError(errQueue <-chan error, handler func(error)) {
	for err := range errQueue {
		if err != nil {
			w.exitcode = 1
			handler(err)
		}
	}
}

func (w *Walker) check(abs string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.checked[abs] {
		return true
	}
	w.checked[abs] = true
	return false
}

func (w *Walker) dirWalker(done <-chan struct{}, errQueue chan<- error) {
	var dir string
	var dirs []string
	var nextDirs []string
	var fis []os.FileInfo
	var err error
	for ; ; w.wg.Done() {
		select {
		case <-done:
			return
		case dirs = <-w.dirQueue:
		NextDirs:
			for _, dir = range dirs {
				if w.check(dir) {
					continue
				}
				fis, err = ioutil.ReadDir(dir)
				if err != nil {
					errQueue <- err
					continue
				}
				for _, fi := range fis {
					if fi.IsDir() {
						nextDirs = append(nextDirs, filepath.Join(dir, fi.Name()))
					} else if fi.Mode().IsRegular() {
						w.wg.Add(1)
						w.fileQueue <- filepath.Join(dir, fi.Name())
					}
				}
			}
			if len(nextDirs) != 0 {
				dirs = append(dirs[:0], nextDirs...)
				nextDirs = nextDirs[:0]
				goto NextDirs
			}
		}
	}
}

// do something for files.
func (w *Walker) fileWalker(done <-chan struct{}, rq chan<- *File, errQueue chan<- error) {
	var file string
	fr := NewFileReader(w.re, w.nbefore, w.nafter)
	var f *File
	var err error
	for ; ; w.wg.Done() {
		select {
		case <-done:
			return
		case file = <-w.fileQueue:
			if w.check(file) {
				continue
			}
			f, err = fr.ReadFile(file)
			if err != nil {
				errQueue <- err
				continue
			}
			rq <- f
		}
	}
}
