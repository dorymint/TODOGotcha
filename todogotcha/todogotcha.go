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
	"strings"
)

const version = "0.0.0rc1"

// exit code
const (
	ErrInitialize = iota + 1
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
	flag.BoolVar(&opt.force, "force", false, "accept overwrite")
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

// TODO: consider
func (g *gotcha) gather(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	index := -1
	lineCount := uint(0)
	addCount := uint(0)

	var push func()
	if g.trim {
		push = func() {
			g.m[path] = append(g.m[path], fmt.Sprintf("L%v:%s", lineCount, sc.Text()[index+len(g.match):]))
			addCount = 1
		}
	} else {
		push = func() {
			g.m[path] = append(g.m[path], fmt.Sprintf("L%v:%s", lineCount, sc.Text()))
			addCount = 1
		}
	}

	var pushNextLines func()
	if g.add != 0 {
		pushNextLines = func() {
			if addCount != 0 && addCount <= g.add {
				g.m[path] = append(g.m[path], fmt.Sprintf(" %v:%s", lineCount, sc.Text()))
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
			return err
		}
		if g.maxRune > 0 && len(sc.Text()) > g.maxRune {
			return fmt.Errorf("have too long line: %v", path)
		}
		if index = strings.Index(sc.Text(), g.match); index != -1 {
			push()
			continue
		}
		pushNextLines()
	}
	return nil
}

func (g *gotcha) crawl() error {
	return filepath.Walk(g.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			g.log.Println(err)
			if g.abort {
				return err
			}
			return nil
		}
		if info.IsDir() && g.ignoreDirsMap[info.Name()] {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && g.isTarget(path) {
			if err := g.gather(path); err != nil {
				g.log.Println(err)
			}
		}
		return nil
	})
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

func (g *gotcha) nContents() uint {
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

func makeBoolMap(list string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range filepath.SplitList(list) {
		m[s] = true
	}
	return m
}

func run(w, errw io.Writer, opt *option) int {
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
		fmt.Fprintln(errw, err)
		return ErrInitialize
	}
	opt.root = fullpath

	g := &gotcha{
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

	if opt.version {
		fmt.Fprintln(w, "todogotcha version "+version)
		return 0
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
	if opt.verbose {
		g.log.SetOutput(errw)
	}

	if err := g.crawl(); err != nil {
		fmt.Fprintln(errw, err)
		return ErrMakeData
	}

	_, err = fmt.Fprintln(w, g)
	if opt.total {
		_, err = fmt.Fprintf(w, "files %d\ncontents %d\n", len(g.m), g.nContents())
	}
	if err != nil {
		fmt.Fprintln(errw, err)
		return ErrOutput
	}
	return 0
}

func main() {
	// TODO: fix
	// test with goroutine
	if true {
		os.Exit(runWithGoroutine(os.Stdout, os.Stderr, opt))
	} else {
		os.Exit(run(os.Stdout, os.Stderr, opt))
	}
}
