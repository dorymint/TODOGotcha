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

// TODO: fix NWorker, use flag?
var NWorker = func() int {
	n := runtime.NumCPU()
	if n <= 0 {
		return 1
	}
	return n
}()

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

func (g *gotcha) gatherWithReturn(path string) *gotchaMap {
	gm := &gotchaMap{path: path}
	var f *os.File
	f, gm.err = os.Open(path)
	if gm.err != nil {
		return gm
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	index := -1
	lineCount := uint(0)
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
	wg := new(sync.WaitGroup)
	queue := make(chan string, 512)
	gatherQueue := make(chan string, 512)
	res := make(chan *gotchaMap, 512)
	errch := make(chan error, 128)

	// error handler
	go func() {
		for {
			// TODO: error handling
			err := <-errch
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
		}
	}()

	// woker
	for i := 0; i < NWorker; i++ {
		go func() {
			for {
				res <- g.gatherWithReturn(<-gatherQueue)
			}
		}()
	}

	// push result
	go func() {
		for {
			// TODO: consider handling of case gm == nil
			gm := <-res
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
		}
	}()

	// walker
	go func() {
		for {
			dir := <-queue
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
		}
	}()

	wg.Add(1)
	queue <- root
	wg.Wait()
	return exitCode
}

func runWithGoroutine(w, errw io.Writer, opt *option) int {
	flag.Parse()
	if flag.NArg() != 0 {
		if flag.NArg() == 1 && opt.root == "" {
			opt.root = flag.Arg(0)
		} else {
			fmt.Fprintln(errw, "unknown arguments: ", flag.Args())
			return ErrInitialize
		}
	}

	fullpath, err := filepath.Abs(opt.root)
	if err != nil {
		return ErrInitialize
	}
	opt.root = fullpath
	g := &gotcha{
		// TODO: consider to delete
		m: make(map[string][]string),

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
	if exitCode := g.workGo(opt.root); exitCode != 0 && g.abort {
		return ErrMakeData
	}
	_, err = fmt.Fprintln(w, g)
	if opt.total {
		_, err = fmt.Fprintf(w, "files %d\ncontents %d\n", len(g.m), func() uint {
			result := uint(0)
			for _, c := range g.m {
				result += uint(len(c))
			}
			return result
		}())
	}
	if err != nil {
		fmt.Fprintln(errw, err)
		return ErrOutput
	}
	return 0
}
