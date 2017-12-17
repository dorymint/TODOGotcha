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
	"runtime"
	"strings"
	"sync"
)

type gotcha struct {
	w   io.Writer
	log *log.Logger

	// options
	root           string
	word           string
	typesMap       map[string]bool
	ignoreDirsMap  map[string]bool
	ignoreFilesMap map[string]bool
	ignoreTypesMap map[string]bool

	// TODO: consider
	maxRune int
	add     uint64
	trim    bool
	abort   bool

	ncontents uint64
	nfiles    uint64
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

// TODO: consider name
type gatherRes struct {
	path     string
	contents []string
	err      error
}

func (gr *gatherRes) Error() string {
	if gr.err == ErrHaveTooLongLine {
		return gr.err.Error() + ":" + gr.path
	}
	return gr.err.Error()
}

// ErrHaveTooLongLine read limit of over
var ErrHaveTooLongLine = errors.New("have too long line")

// IsTooLong check ErrHaveTooLongLine
func IsTooLong(err error) bool {
	switch e := err.(type) {
	case *gatherRes:
		return e.err == ErrHaveTooLongLine
	}
	return false
}

func (g *gotcha) gather(path string) *gatherRes {
	gr := &gatherRes{path: path}
	var f *os.File
	f, gr.err = os.Open(path)
	if gr.err != nil {
		return gr
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	index := -1
	lineCount := uint64(1) // TODO: consider to zero
	addCount := uint64(0)

	var push func()
	if g.trim {
		push = func() {
			gr.contents = append(gr.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.word):]))
			addCount = 1
		}
	} else {
		push = func() {
			gr.contents = append(gr.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()))
			addCount = 1
		}
	}

	var pushNextLines func()
	if g.add != 0 {
		pushNextLines = func() {
			if addCount != 0 && addCount <= g.add {
				gr.contents = append(gr.contents, fmt.Sprintf(" %v:%s", lineCount, sc.Text()))
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
		if gr.err = sc.Err(); gr.err != nil {
			return gr
		}
		if g.maxRune > 0 && len(sc.Text()) > g.maxRune {
			gr.err = ErrHaveTooLongLine
			return gr
		}
		if index = strings.Index(sc.Text(), g.word); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return gr
}

// TODO: implementation syncWorkGo

// make map on async
func (g *gotcha) WorkGo(root string) (exitCode int) {
	// queue -> gatherQueue -> res
	var (
		wg          = new(sync.WaitGroup)
		queue       = make(chan string, 512)
		gatherQueue = make(chan string, 512)
		res         = make(chan *gatherRes, 512)
		errch       = make(chan error, 128)
	)

	// TODO: consider really need? goCounter
	var (
		goCounter = uint64(0)
		done      = make(chan bool)
	)
	defer func() {
		for ; goCounter != 0; goCounter-- {
			done <- true
		}
	}()

	// TODO: addWorker use flag?
	addWorker := func() uint64 {
		n := runtime.NumCPU() / 2
		if n < 0 {
			return 0
		}
		return uint64(n)
	}()

	// error handler
	// TODO: consider error handling
	//     : this is maybe discard some errors
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
						g.log.Fatal(err) // TODO: consider not use panic
					case IsTooLong(err), os.IsPermission(err), os.IsNotExist(err):
						g.log.Println(err)
						continue
					default:
						g.log.Fatalln("unknown error:", err)
						//panic(err) // TODO: consider not use panic
					}
				}
			case <-done:
				return
			}
		}
	}()

	// woker
	for i := uint64(0); i <= addWorker; i++ {
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

	// res with write
	goCounter++
	go func() {
		for {
			select {
			case gr := <-res:
				switch {
				case gr.err != nil:
					if gr.err == ErrHaveTooLongLine {
						errch <- gr
					} else {
						errch <- gr.err
					}
				case len(gr.contents) != 0:
					_, err := fmt.Fprintln(g.w, gr.path+"\n"+strings.Join(gr.contents, "\n")+"\n")
					if err != nil {
						errch <- err
					}
					g.nfiles++
					g.ncontents += uint64(len(gr.contents))
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
						// TODO: consider another way
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

// TODO: consider SyncWorkGo
func (g *gotcha) SyncWorkGo(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && g.ignoreDirsMap[info.Name()] {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && g.isTarget(info.Name()) {
			gr := g.gather(path)
			if gr.err != nil {
				switch {
				case g.abort:
					g.log.Fatal(gr.err) // TODO: consider not use panic
				case IsTooLong(gr):
					g.log.Print(gr)
				case os.IsPermission(gr.err) || os.IsNotExist(gr.err):
					g.log.Print(gr.err)
				default:
					g.log.Print(gr.err) // TODO: consider not use panic
				}
				return nil
			}
			if len(gr.contents) != 0 {
				_, err = fmt.Fprintln(g.w, gr.path+"\n"+strings.Join(gr.contents, "\n")+"\n")
				g.nfiles++
				g.ncontents += uint64(len(gr.contents))
			}
		}
		return err
	})
}
