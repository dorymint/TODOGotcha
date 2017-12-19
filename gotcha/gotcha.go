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

// Gotcha for search recursive
type Gotcha struct {
	W   io.Writer
	Log *log.Logger

	// options
	Word           string
	TypesMap       map[string]bool
	IgnoreDirsMap  map[string]bool
	IgnoreBasesMap map[string]bool
	IgnoreTypesMap map[string]bool

	// TODO: consider
	MaxRune int
	Add     uint64
	Trim    bool
	Abort   bool

	nfiles uint64
	nlines uint64
}

// NewGotcha allocation for Gotcha
func NewGotcha() *Gotcha {
	makeBoolMap := func(list []string) map[string]bool {
		m := make(map[string]bool)
		for _, s := range list {
			m[s] = true
		}
		return m
	}
	return &Gotcha{
		W:   os.Stdout,
		Log: log.New(ioutil.Discard, "["+name+"]:", log.Lshortfile),

		Word:           "TODO: ",
		TypesMap:       make(map[string]bool),
		IgnoreDirsMap:  makeBoolMap(IgnoreDirs),
		IgnoreBasesMap: makeBoolMap(IgnoreBases),
		IgnoreTypesMap: makeBoolMap(IgnoreTypes),

		MaxRune: 512,
		Add:     0,
		Trim:    false,
		Abort:   false,

		nfiles: 0,
		nlines: 0,
	}
}

// PrintTotal prnt nfiles and ncontents
func (g *Gotcha) PrintTotal() (int, error) {
	return fmt.Fprintf(g.W, "files %d\nlines %d\n", g.nfiles, g.nlines)
}

func (g *Gotcha) isTarget(path string) bool {
	if g.IgnoreBasesMap[path] {
		return false
	}
	ext := filepath.Ext(path)
	if g.IgnoreTypesMap[ext] {
		return false
	}
	if len(g.TypesMap) == 0 {
		return true
	}
	return g.TypesMap[ext]
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

func (g *Gotcha) gather(path string) *gatherRes {
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
	if g.Trim {
		push = func() {
			gr.contents = append(gr.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.Word):]))
			addCount = 1
		}
	} else {
		push = func() {
			gr.contents = append(gr.contents, fmt.Sprintf("L%v:%s", lineCount, sc.Text()))
			addCount = 1
		}
	}

	var pushNextLines func()
	if g.Add != 0 {
		pushNextLines = func() {
			if addCount != 0 && addCount <= g.Add {
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
		if g.MaxRune > 0 && len(sc.Text()) > g.MaxRune {
			gr.err = ErrHaveTooLongLine
			return gr
		}
		if index = strings.Index(sc.Text(), g.Word); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return gr
}

// WorkGo run on async
func (g *Gotcha) WorkGo(root string, nworker uint64) (exitCode int) {
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

	// TODO: consider
	if nworker == 0 {
		nworker = func() uint64 {
			n := runtime.NumCPU() / 2
			if n < 0 {
				return 0
			}
			return uint64(n)
		}()
	}

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
					case g.Abort:
						g.Log.Fatal(err) // TODO: consider
					case IsTooLong(err), os.IsPermission(err), os.IsNotExist(err):
						g.Log.Println(err)
						continue
					default:
						g.Log.Fatalln("unknown error:", err)
						//panic(err) // TODO: consider
					}
				}
			case <-done:
				return
			}
		}
	}()

	// woker
	for i := uint64(0); i <= nworker; i++ {
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
					_, err := fmt.Fprintf(g.W, "%s\n%s\n\n", gr.path, strings.Join(gr.contents, "\n"))
					if err != nil {
						errch <- err
					}
					g.nfiles++
					g.nlines += uint64(len(gr.contents))
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
					case info.IsDir() && !g.IgnoreDirsMap[info.Name()]:
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

// SyncWorkGo run on sync
func (g *Gotcha) SyncWorkGo(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && g.IgnoreDirsMap[info.Name()] {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && g.isTarget(info.Name()) {
			gr := g.gather(path)
			if gr.err != nil {
				switch {
				case g.Abort:
					g.Log.Fatal(gr.err) // TODO: consider not use panic
				case IsTooLong(gr):
					g.Log.Print(gr)
				case os.IsPermission(gr.err) || os.IsNotExist(gr.err):
					g.Log.Print(gr.err)
				default:
					g.Log.Print(gr.err) // TODO: consider not use panic
				}
				return nil
			}
			if len(gr.contents) != 0 {
				_, err = fmt.Fprintf(g.W, "%s\n%s\n\n", gr.path, strings.Join(gr.contents, "\n"))
				g.nfiles++
				g.nlines += uint64(len(gr.contents))
			}
		}
		return err
	})
}
