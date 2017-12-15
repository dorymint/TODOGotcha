package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const version = "0.0.1rc1"

// exit code
const (
	ValidExit = iota
	ErrInitialize
	ErrMakeData
	ErrOutput
)

type option struct {
	version bool
	root    string
	word    string
	verbose bool
	abort   bool
	out     string
	force   bool
	total   bool

	ftypes      string
	ignoreDirs  string
	ignoreFiles string
	ignoreTypes string

	trim bool
	add  uint

	maxRune int
}

var opt = &option{}

// Default Ignores
var (
	IgnoreDirs = []string{
		".git",
		".cache",
	}
	IgnoreFiles = []string{}
	IgnoreTypes = []string{
		".pgp", ".ttf", ".pdf",
		".jpg", ".jpeg", ".png", ".ico", ".gif",
		".mp4",
		".mp3", ".ogg", ".wav", ".au",
		".so", ".mo", ".a", ".o", ".pyc", ".exe", ".efi",
		".gz", ".xz", ".tar", ".bz2", ".db", ".tgz", ".zip",
	}
)

func init() {
	flag.BoolVar(&opt.version, "version", false, "print version")
	flag.StringVar(&opt.root, "root", "", "specify search root directory")
	flag.StringVar(&opt.word, "word", "TODO: ", "specify search word")
	flag.StringVar(&opt.out, "out", "", "specify output file")
	flag.BoolVar(&opt.force, "force", false, "accept overwrite for \"-out\"")
	flag.BoolVar(&opt.total, "total", false, "prints total number of contents")

	sep := string(filepath.ListSeparator)
	flag.StringVar(&opt.ftypes, "types", "", "specify filetypes. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreDirs, "ignore-dirs", strings.Join(IgnoreDirs, sep), "specify ignore directories. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreFiles, "ignore-files", strings.Join(IgnoreFiles, sep), "specify ignore files. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreTypes, "ignore-types", strings.Join(IgnoreTypes, sep), "specify ignore file types. separator is '"+sep+"'")

	flag.BoolVar(&opt.trim, "trim", false, "trim the word on output")
	flag.UintVar(&opt.add, "add", 0, "specify number of lines of after find the word")

	flag.IntVar(&opt.maxRune, "max", 512, "specify characters limit")
	flag.BoolVar(&opt.verbose, "verbose", false, "output of log messages")
	flag.BoolVar(&opt.abort, "abort", false, "if exists errors then abort process")
}

type gotcha struct {
	m map[string][]string

	// options
	root           string
	match          string
	typesMap       map[string]bool
	ignoreDirsMap  map[string]bool
	ignoreFilesMap map[string]bool
	ignoreTypesMap map[string]bool

	// TODO: consider
	maxRune int
	add     uint
	trim    bool
	abort   bool

	log *log.Logger
}

func (g *gotcha) isTarget(path string) bool {
	if g.ignoreFilesMap[path] {
		return false
	}
	ext := filepath.Ext(path)
	if g.ignoreTypesMap[ext] {
		return false
	}
	if len(g.typesMap) == 0 {
		return true
	}
	return g.typesMap[ext]
}

func (g *gotcha) NContents() uint {
	ui := uint(0)
	for _, c := range g.m {
		ui += uint(len(c))
	}
	return ui
}

// consider String()
func (g *gotcha) String() string {
	var result string
	for path, strs := range g.m {
		if len(strs) == 0 {
			continue
		}
		result += fmt.Sprintln(path)
		for _, str := range strs {
			result += fmt.Sprintln(str)
		}
		result += fmt.Sprintln()
	}
	return result
}

// TODO: consider name
type gotchaMap struct {
	path     string
	contents []string
	err      error
}

func (gm *gotchaMap) Error() string { return fmt.Sprintf("%v:%v", gm.err.Error(), gm.path) }

// ErrHaveTooLongLine read limit of over
var ErrHaveTooLongLine = errors.New("have too long line")

func isTooLong(err error) bool {
	switch err.(type) {
	case *gotchaMap:
		return true
	}
	return false
}

func (g *gotcha) gather(path string) *gotchaMap {
	gm := &gotchaMap{path: path}
	var f *os.File
	f, gm.err = os.Open(path)
	if gm.err != nil {
		return gm
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	index := -1
	lineCount := uint(1) // TODO: consider to zero
	addCount := uint(0)

	var push func()
	if g.trim {
		push = func() {
			gm.contents = append(gm.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.match):]))
			addCount = 1
		}
	} else {
		push = func() {
			gm.contents = append(gm.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()))
			addCount = 1
		}
	}

	var pushNextLines func()
	if g.add != 0 {
		pushNextLines = func() {
			if addCount != 0 && addCount <= g.add {
				gm.contents = append(gm.contents, fmt.Sprintf(" %v:%s", lineCount, sc.Text()))
				addCount++
			} else {
				addCount = 0
			}
		}
	} else {
		// discard
		pushNextLines = func() {}
	}

	for ; sc.Scan(); lineCount++ {
		if gm.err = sc.Err(); gm.err != nil {
			return gm
		}
		if g.maxRune > 0 && len(sc.Text()) > g.maxRune {
			gm.err = ErrHaveTooLongLine
			return gm
		}
		if index = strings.Index(sc.Text(), g.match); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return gm
}

func (g *gotcha) workGo(root string) (exitCode int) {
	var (
		wg          = new(sync.WaitGroup)
		queue       = make(chan string, 512)
		gatherQueue = make(chan string, 512)
		res         = make(chan *gotchaMap, 512)
		errch       = make(chan error, 128)
	)

	// TODO: consider really need? goCounter
	var (
		goCounter = uint(0)
		done      = make(chan bool)
	)
	defer func() {
		for ; goCounter != 0; goCounter-- {
			done <- true
		}
	}()

	// TODO: nWorker use flag?
	var nWorker = func() int {
		n := runtime.NumCPU()
		// really need?
		if n <= 0 {
			return 1
		}
		return n
	}()

	// TODO: consider error handling
	//     : maybe discard some errors
	// error handler
	goCounter++
	go func() {
		for {
			select {
			case err := <-errch:
				// TODO: error handling
				if err != nil {
					exitCode = 1 // TODO: consider exitCode
					switch {
					case g.abort:
						panic(err) // TODO: consider not use panic
					case os.IsPermission(err) || os.IsNotExist(err) || isTooLong(err):
						g.log.Println(err)
						continue
					default:
						panic(err) // TODO: consider not use panic
					}
				}
			case <-done:
				return
			}
		}
	}()

	// woker
	for i := 0; i < nWorker; i++ {
		goCounter++
		go func() {
			for {
				select {
				case path := <-gatherQueue:
					res <- g.gather(path)
				case <-done:
					return
				}
			}
		}()
	}

	// push result
	goCounter++
	go func() {
		for {
			select {
			case gm := <-res:
				// TODO: consider handling of case gm == nil
				if gm == nil {
					wg.Done()
					continue
				}
				if gm.err != nil {
					errch <- gm
				}
				if len(gm.contents) != 0 {
					g.m[gm.path] = gm.contents
				}
				wg.Done()
			case <-done:
				return
			}
		}
	}()

	// walker
	goCounter++
	go func() {
		for {
			select {
			case dir := <-queue:
				infos, err := ioutil.ReadDir(dir)
				if err != nil {
					errch <- err
					wg.Done()
					continue
				}
				for _, info := range infos {
					path := filepath.Join(dir, info.Name())
					switch {
					case info.IsDir() && !g.ignoreDirsMap[info.Name()]:
						wg.Add(1)
						go func(path string) { queue <- path }(path)
					case info.Mode().IsRegular() && g.isTarget(info.Name()):
						wg.Add(1)
						gatherQueue <- path
					}
				}
				wg.Done()
			case <-done:
				return
			}
		}
	}()

	wg.Add(1)
	queue <- root
	wg.Wait()
	return exitCode
}

func run(w, errw io.Writer, opt *option) int {
	if opt.version {
		fmt.Fprintln(w, "todogotha version "+version)
		return ValidExit
	}

	fullpath, err := filepath.Abs(opt.root)
	if err != nil {
		fmt.Fprintln(errw, err)
		return ErrInitialize
	}
	opt.root = fullpath

	makeBoolMap := func(list string) map[string]bool {
		m := make(map[string]bool)
		for _, s := range filepath.SplitList(list) {
			m[s] = true
		}
		return m
	}
	g := &gotcha{
		m:              make(map[string][]string),
		root:           opt.root,
		match:          opt.word,
		abort:          opt.abort,
		typesMap:       makeBoolMap(opt.ftypes),
		ignoreDirsMap:  makeBoolMap(opt.ignoreDirs),
		ignoreFilesMap: makeBoolMap(opt.ignoreFiles),
		ignoreTypesMap: makeBoolMap(opt.ignoreTypes),

		maxRune: opt.maxRune,
		add:     opt.add,

		log: log.New(ioutil.Discard, "[todogotcha]:", log.Lshortfile),
	}

	if opt.verbose {
		g.log.SetOutput(errw)
	}

	if opt.out != "" {
		if _, err := os.Stat(opt.out); os.IsExist(err) && !opt.force {
			fmt.Fprintln(errw, "file exists: ", opt.out)
			return ErrInitialize
		}
		f, err := os.Create(opt.out)
		if err != nil {
			fmt.Fprintln(errw, err)
			return ErrInitialize
		}
		defer f.Close()
		w = f
	}

	// TODO: fix exitCode
	if exitCode := g.workGo(opt.root); exitCode != ValidExit && g.abort {
		return ErrMakeData
	}

	_, err = fmt.Fprint(w, g)
	if opt.total {
		_, err = fmt.Fprintf(w, "files %d\ncontents %d\n", len(g.m), g.NContents())
	}
	if err != nil {
		fmt.Fprintln(errw, err)
		return ErrOutput
	}
	return ValidExit
}

func main() {
	flag.Parse()
	if flag.NArg() != 0 {
		if flag.NArg() == 1 && opt.root == "" {
			opt.root = flag.Arg(0)
		} else {
			fmt.Fprintln(os.Stderr, "unknown arguments: ", flag.Args())
			os.Exit(ErrInitialize)
		}
	}
	os.Exit(run(os.Stdout, os.Stderr, opt))
}
