package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	version = "0.4.0rc3"
	name    = "gotcha"
)

// exit code
const (
	ValidExit = iota
	ErrInitialize
	ErrRun
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

	// specify target file types
	types string

	// ignores
	ignoreDirs  string
	ignoreBases string
	ignoreTypes string

	trim bool
	add  uint64

	maxRune int

	nworker uint64
	sync    bool
	cache   bool
}

var opt = &option{}

// TODO: consider default ignores
// Default Ignores
var (
	IgnoreDirs = []string{
		".git",
		".cache",
	}
	IgnoreBases = []string{}
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
	flag.BoolVar(&opt.version, "version", false, "print version "+`"`+version+`"`)
	flag.StringVar(&opt.root, "root", "", "specify search root directory")
	flag.StringVar(&opt.word, "word", "TODO: ", "specify search word")
	flag.StringVar(&opt.out, "out", "", "specify output file")
	flag.BoolVar(&opt.force, "force", false, "accept overwrite for \"-out\"")
	flag.BoolVar(&opt.total, "total", false, "prints total number of contents")

	sep := string(filepath.ListSeparator)
	flag.StringVar(&opt.types, "types", "", "specify filetypes. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreDirs, "ignore-dirs", strings.Join(IgnoreDirs, sep), "specify ignore directories. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreBases, "ignore-bases", strings.Join(IgnoreBases, sep), "specify ignore basenames. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreTypes, "ignore-types", strings.Join(IgnoreTypes, sep), "specify ignore file types. separator is '"+sep+"'")

	flag.BoolVar(&opt.trim, "trim", false, "trim the word on output")
	flag.Uint64Var(&opt.add, "add", 0, "specify number of lines of after find the word")

	flag.IntVar(&opt.maxRune, "max", 256, "specify characters limit")
	flag.BoolVar(&opt.verbose, "verbose", false, "output of log messages")
	flag.BoolVar(&opt.abort, "abort", false, "if exists errors then abort process")

	flag.Uint64Var(&opt.nworker, "nworker", 0, "specify limitation of goriutine")
	flag.BoolVar(&opt.sync, "sync", false, "for debug: run on sync")
	flag.BoolVar(&opt.cache, "cache", false, "use data cache")
}

func run(w, errw io.Writer, opt *option) (exitCode int) {
	// version
	if opt.version {
		fmt.Fprintln(w, name+" version "+version)
		return
	}

	// abs for root
	fullpath, err := filepath.Abs(opt.root)
	if err != nil {
		fmt.Fprintln(errw, err)
		exitCode = ErrInitialize
		return
	}
	opt.root = fullpath

	// out to file
	if opt.out != "" {
		if _, err := os.Stat(opt.out); os.IsExist(err) && !opt.force {
			fmt.Fprintln(errw, "file exists: ", opt.out)
			exitCode = ErrInitialize
			return
		}
		f, err := os.Create(opt.out)
		if err != nil {
			fmt.Fprintln(errw, err)
			exitCode = ErrInitialize
			return
		}
		defer f.Close()
		w = f
	}

	// use buffer
	if opt.cache {
		orgiw := w
		buf := bytes.NewBufferString("")
		w = buf
		defer func() {
			_, err := fmt.Fprintln(orgiw, w)
			if err != nil {
				fmt.Fprintln(errw, err)
				exitCode = ErrRun
			}
		}()
	}

	/// init Gotcha
	makeBoolMap := func(list string) map[string]bool {
		m := make(map[string]bool)
		for _, s := range filepath.SplitList(list) {
			m[s] = true
		}
		return m
	}
	g := NewGotcha()
	g.W = w
	g.Word = opt.word
	g.Abort = opt.abort
	g.TypesMap = makeBoolMap(opt.types)
	g.IgnoreDirsMap = makeBoolMap(opt.ignoreDirs)
	g.IgnoreBasesMap = makeBoolMap(opt.ignoreBases)
	g.IgnoreTypesMap = makeBoolMap(opt.ignoreTypes)
	g.MaxRune = opt.maxRune
	g.Add = opt.add
	if opt.verbose {
		g.Log.SetOutput(errw)
	}

	// sync or async
	if opt.sync {
		err := g.SyncWorkGo(opt.root)
		if err != nil {
			fmt.Fprint(errw, err)
			exitCode = ErrRun
		}
	} else {
		exitCode = g.WorkGo(opt.root, opt.nworker)
	}

	// append total
	if opt.total {
		_, err = g.PrintTotal()
		if err != nil {
			fmt.Fprint(errw, err)
			exitCode = ErrRun
		}
	}
	return
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
