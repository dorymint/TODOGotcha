package main

import (
	"bufio"
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

// TODO: fix
var Ngoroutine = func() int {
	n := runtime.NumCPU()
	if n <= 0 {
		return 1
	}
	return n
}()

// TODO: consider need
type gotchaMap struct {
	path     string
	contents []string
}

func (g *gotcha) gatherWithReturn(path string) (*gotchaMap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &gotchaMap{path: path}

	sc := bufio.NewScanner(f)
	index := -1
	lineCount := uint(0)
	addCount := uint(0)

	var push func()
	if g.trim {
		push = func() {
			result.contents = append(result.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.match):]))
			addCount = 1
		}
	} else {
		push = func() {
			result.contents = append(result.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()))
			addCount = 1
		}
	}

	var pushNextLines func()
	if g.add != 0 {
		pushNextLines = func() {
			if addCount != 0 && addCount <= g.add {
				result.contents = append(result.contents, fmt.Sprintf(" %v:%s", lineCount, sc.Text()))
				addCount++
			} else {
				addCount = 0
			}
		}
	} else {
		pushNextLines = func() {}
	}

	for ; sc.Scan(); lineCount++ {
		if err := sc.Err(); err != nil {
			return nil, err
		}
		if g.maxRune > 0 && len(sc.Text()) > g.maxRune {
			return nil, fmt.Errorf("have too long line: %v", path)
		}
		if index = strings.Index(sc.Text(), g.match); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return result, nil
}

func (g *gotcha) walk(root string) error {
	wg, queue := g.workGo()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			switch {
			case g.abort:
				return err
			case os.IsPermission(err) || os.IsNotExist(err):
				return nil
			default:
				return err
			}
		}
		if info.IsDir() && g.ignoreDirsMap[info.Name()] {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && g.isTarget(info.Name()) {
			wg.Add(1)
			queue <- path
		}
		return nil
	})
	wg.Wait()
	return err
}

func (g *gotcha) workGo() (*sync.WaitGroup, chan<- string) {
	wg := new(sync.WaitGroup)
	queue := make(chan string, 512)
	res := make(chan *gotchaMap, 512)
	errch := make(chan error, 128)

	// error handler
	go func() {
		for {
			// TODO: error handling
			err := <-errch
			g.log.Println(err)
		}
	}()

	// woker
	for i := 0; i < Ngoroutine; i++ {
		go func() {
			for {
				gm, err := g.gatherWithReturn(<-queue)
				errch <- err
				res <- gm
			}
		}()
	}

	// push result
	go func() {
		for {
			if result := <-res; result != nil && len(result.contents) != 0 {
				g.m[result.path] = result.contents
			}
			wg.Done()
		}
	}()
	return wg, queue
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

	if err := g.walk(opt.root); err != nil {
		fmt.Fprintln(errw, err)
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
