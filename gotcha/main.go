package main

// TODO: to simpl

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// version and cmd name
const (
	Version = "0.1.0dev"
	Name    = "gotcha"
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
	add  uint

	maxRune int

	nworker uint
	sync    bool
	cache   bool

	verbose bool
}

var opt = &option{}

// TODO: consider default ignores
// Default Ignores
var (
	IgnoreDirs = []string{
		".git",
		".cache",
	}
	// TODO: implementation to read .gotcha
	IgnoreBases = []string{
		".gotcha",
	}
	IgnoreTypes = []string{
		".iso", ".img",
		".log", ".prof",
		".pgp", ".ttf", ".pdf",
		".jpg", ".jpeg", ".png", ".ico", ".gif",
		".mp4",
		".mp3", ".ogg", ".wav", ".au",
		".so", ".mo", ".a", ".o", ".pyc", ".exe", ".efi",
		".gz", ".xz", ".tar", ".bz", ".bz2", ".db", ".tgz", ".zip",
	}
)

func init() {
	flag.BoolVar(&opt.version, "version", false, "print version "+`"`+Version+`"`)
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
	flag.UintVar(&opt.add, "add", 0, "specify number of lines of after find the word")

	flag.IntVar(&opt.maxRune, "max", 256, "specify characters limit")
	flag.BoolVar(&opt.abort, "abort", false, "if exists errors then abort process")

	flag.UintVar(&opt.nworker, "nworker", 0, "specify limitation of gather worker")
	flag.BoolVar(&opt.sync, "sync", false, "for debug: run on sync")
	flag.BoolVar(&opt.cache, "cache", false, "use data cache")

	flag.BoolVar(&opt.verbose, "verbose", false, "verbose output")
}

func run(w, errw io.Writer, opt *option) (exitCode int) {
	// version
	if opt.version {
		fmt.Fprintln(w, Name+" version "+Version)
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
		if _, err := os.Stat(opt.out); err == nil && !opt.force {
			fmt.Fprintln(errw, "exists: ", opt.out)
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
		origw := w
		origerrw := errw
		buf := bytes.NewBufferString("")
		errbuf := bytes.NewBufferString("")
		w = buf
		errw = errbuf
		defer func() {
			_, err := io.Copy(origw, buf)
			if err != nil {
				fmt.Fprintln(errw, err)
				exitCode = ErrRun
			}
			io.Copy(origerrw, errbuf)
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
	} else {
		g.Log.SetOutput(ioutil.Discard)
	}

	/// TODO: case opt.root == ToFilePath:
	info, err := os.Stat(opt.root)
	if err != nil {
		fmt.Fprintln(errw, err)
		exitCode = ErrInitialize
		return
	}
	switch {
	case info.IsDir():
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
	case info.Mode().IsRegular():
		res := g.gather(opt.root)
		if res.err != nil {
			fmt.Fprintln(errw, res)
			exitCode = ErrRun
		}
		if err := res.Fwrite(g.W); err != nil {
			fmt.Fprintln(errw, err)
			exitCode = ErrRun
		}
	default:
		fmt.Fprintln(errw, "invalid file type")
		exitCode = ErrRun
	}

	// append total
	if opt.total {
		_, err = g.PrintTotal()
		if err != nil {
			fmt.Fprint(errw, err)
			exitCode = ErrRun
		}
	}
	return exitCode
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
