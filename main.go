// rgr is cli tools for find target words in many files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
)

const (
	Version = "0.5.1"
	Name    = "rgr"
)

var usageWriter io.Writer = os.Stderr

const usage = `Usage:
  rgr [Options]
  rgr -- STRING
  rgr -- STRING [PATH...]

Options:
  -help              Print this help
  -version           Print version
  -verbose           Verbose output
  -e, -regexp        Use regexp
  -C, -context [Num] With context
  -A, -after   [Num] Specify after lines
  -B, -before  [Num] Specify before lines

Examples:
  # search "func"
  $ rgr "func" main.go vendor/

  # search recursively from current directory
  $ rgr "func"

  # with context
  $ rgr -C 3 "func" main.go vendor/
`

func printUsage() {
	_, err := fmt.Fprint(usageWriter, usage)
	if err != nil {
		panic(err)
	}
}

var opt struct {
	help    bool
	version bool

	verbose bool
	regexp  bool

	// TODO?
	// %f
	// %l:%m
	//
	// change to
	// %f:%l:%c:%m
	//
	//style string

	context int
	before  int
	after   int
}

func init() {
	flag.BoolVar(&opt.help, "help", false, "Print usage")
	flag.BoolVar(&opt.version, "version", false, "Print version")

	flag.BoolVar(&opt.verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&opt.regexp, "regexp", false, "Use regexp")
	flag.BoolVar(&opt.regexp, "e", false, "Alias of -regexp")

	flag.IntVar(&opt.context, "context", 0, "Append context")
	flag.IntVar(&opt.context, "C", 0, "Alias of -context")

	flag.IntVar(&opt.before, "before", 0, "Append context")
	flag.IntVar(&opt.before, "B", 0, "Alias of -before")

	flag.IntVar(&opt.after, "after", 0, "Alias of -context")
	flag.IntVar(&opt.after, "A", 0, "Alias of -after")
}

func run() (err error) {
	flag.Usage = printUsage
	flag.Parse()
	switch {
	case opt.help:
		usageWriter = os.Stdout
		flag.Usage()
		return nil
	case opt.version:
		_, err = fmt.Printf("%s %s\n", Name, Version)
		return err
	}
	if flag.NArg() == 0 {
		flag.Usage()
		return errors.New("arguments not enough")
	}

	walker := NewWalker()

	pat := flag.Arg(0)
	if !opt.regexp {
		pat = regexp.QuoteMeta(pat)
	}
	if err = walker.SetRegexp(pat); err != nil {
		return err
	}

	if opt.before == 0 {
		opt.before = opt.context
	}
	if opt.after == 0 {
		opt.after = opt.context
	}
	if opt.before < 0 || opt.after < 0 {
		return errors.New("can not specify negative number")
	}
	if err = walker.SetContext(opt.before, opt.after); err != nil {
		return err
	}

	var rwm sync.RWMutex
	if opt.verbose {
		err = walker.SetErrorHandler(func(err error) {
			rwm.Lock()
			fmt.Fprintln(os.Stderr, err)
			rwm.Unlock()
			DefaultErrorHandler(err)
		})
		if err != nil {
			return err
		}
	}

	fileQueue, wait := walker.Start()

	paths := flag.Args()[1:]
	if len(paths) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		paths = append(paths, pwd)
	}
	if err = walker.SendPath(paths...); err != nil {
		return err
	}

	go wait()
	var f *File
	var c *Context
	for f = range fileQueue {
		if len(f.Contexts) == 0 {
			continue
		}
		rwm.Lock()
		fmt.Println(f.Path)
		for _, c = range f.Contexts {
			fmt.Print(c)
		}
		fmt.Println()
		rwm.Unlock()
	}

	if walker.WaitExitCode() != 0 {
		return errors.New("internal error")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s:[Err]:%v\n", Name, err)
		os.Exit(1)
	}
}
