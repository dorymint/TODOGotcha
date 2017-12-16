package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.0rc1"

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

	types       string
	ignoreDirs  string
	ignoreFiles string
	ignoreTypes string

	trim bool
	add  uint

	maxRune int

	// debug
	sync bool
}

var opt = &option{}

// TODO: consider default ignores
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
	flag.StringVar(&opt.types, "types", "", "specify filetypes. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreDirs, "ignore-dirs", strings.Join(IgnoreDirs, sep), "specify ignore directories. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreFiles, "ignore-files", strings.Join(IgnoreFiles, sep), "specify ignore files. separator is '"+sep+"'")
	flag.StringVar(&opt.ignoreTypes, "ignore-types", strings.Join(IgnoreTypes, sep), "specify ignore file types. separator is '"+sep+"'")

	flag.BoolVar(&opt.trim, "trim", false, "trim the word on output")
	flag.UintVar(&opt.add, "add", 0, "specify number of lines of after find the word")

	flag.IntVar(&opt.maxRune, "max", 512, "specify characters limit")
	flag.BoolVar(&opt.verbose, "verbose", false, "output of log messages")
	flag.BoolVar(&opt.abort, "abort", false, "if exists errors then abort process")

	/// debug
	flag.BoolVar(&opt.sync, "sync", false, "for debug: run on sync")
}

func run(w, errw io.Writer, opt *option) int {
	if opt.version {
		fmt.Fprintln(w, "todogotcha version "+version)
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
		word:           opt.word,
		abort:          opt.abort,
		typesMap:       makeBoolMap(opt.types),
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

	// TODO: consider sync, async and sort
	if opt.sync {
		err := g.syncWorkGo(opt.root)
		if err != nil && !os.IsPermission(err) && !os.IsNotExist(err) && isTooLong(err) {
			switch {
			case os.IsPermission(err), os.IsNotExist(err), isTooLong(err):
				// pass
			default:
				fmt.Fprintln(errw, err)
				return ErrMakeData
			}
		}
	} else {
		exitCode := g.WorkGo(opt.root)
		if exitCode != ValidExit && g.abort {
			return ErrMakeData
		}
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
