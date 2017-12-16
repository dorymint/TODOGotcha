package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type gotcha struct {
	m map[string][]string

	// options
	root           string
	word           string
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
			gm.contents = append(gm.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.word):]))
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
		if index = strings.Index(sc.Text(), g.word); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return gm
}

// TODO: implementation syncWorkGo

// make map on async
func (g *gotcha) WorkGo(root string) (exitCode int) {
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

	// TODO: addWorker use flag?
	addWorker := func() uint {
		n := runtime.NumCPU() / 2
		if n < 0 {
			return 0
		}
		return uint(n)
	}()

	// TODO: consider error handling
	//     : this is maybe discard some errors
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
	for i := uint(0); i <= addWorker; i++ {
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
				} else {
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
						continue
					case info.Mode().IsRegular() && g.isTarget(info.Name()):
						wg.Add(1)
						gatherQueue <- path
						continue
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

// TODO: consider syncWorkGo
func (g *gotcha) syncWorkGo(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && g.ignoreDirsMap[info.Name()] {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && g.isTarget(info.Name()) {
			gm := g.gather(path)
			if gm.err != nil {
				switch {
				case g.abort:
					panic(gm.err) // TODO: consider not use panic
				case isTooLong(gm):
					g.log.Println(gm)
					return nil
				case os.IsPermission(gm.err) || os.IsNotExist(gm.err):
					g.log.Println(gm.err)
					return nil
				default:
					panic(gm) // TODO: consider not use panic
				}
			} else {
				g.m[gm.path] = gm.contents
			}
		}
		return nil
	})
}
