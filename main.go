// rgr is cli tool for recursive search with regular expression.
// available in utf8.
package main

// TODO: fix dpulicate?
// rgr "main" . $(pwd)

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	Version = "0.4.0"
	Name    = "rgr"
)

var usageWriter io.Writer = os.Stderr

const usage = `Usage:
  rgr [Options]
  rgr -- [Regexp]
  rgr -- [Regexp] [Path...]

Options:
  -help              Print this help
  -version           Print version
  -c, -context [Num] With context

  TODO: impl
  -a, -after   [Num] Specify after lines
  -b, -before  [Num] Specify before lines

Examples:
  # search "func"
  $ rgr "func" main.go vendor/

  # search from current directory
  $ rgr "func"

  # with context
  $ rgr -c 3 "func" main.go vendor/
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

	// TODO?
	// Default:
	// %f
	// %l:%m ...
	// Verbose:
	// %f:%l:%c:%m
	style string

	context uint

	// TODO
	before uint
	after  uint

	sync    bool
	verbose bool
}

func init() {
	flag.BoolVar(&opt.help, "help", false, "Print usage")
	flag.BoolVar(&opt.version, "version", false, "Print version")

	flag.UintVar(&opt.context, "context", 0, "Append context")
	flag.UintVar(&opt.context, "c", 0, "Alias of -context")

	flag.BoolVar(&opt.sync, "sync", false, "Enable sync output")
	flag.BoolVar(&opt.verbose, "verbose", false, "Verbose output")
	flag.Usage = printUsage
	flag.Parse()
}

func run() error {
	switch {
	case opt.help:
		usageWriter = os.Stdout
		flag.Usage()
		return nil
	case opt.version:
		_, err := fmt.Printf("%s %s\n", Name, Version)
		return err
	}
	if flag.NArg() == 0 {
		flag.Usage()
		return errors.New("arguments not enough")
	}
	pat := flag.Arg(0)
	paths := flag.Args()[1:]
	if len(paths) == 0 {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		paths = append(paths, pwd)
	}

	walker := NewWalker()
	if opt.verbose {
		walker.SetLogOutput(os.Stderr)
	}
	fch, wait, err := walker.Start(pat, opt.context, paths...)
	if err != nil {
		return err
	}

	if opt.sync {
		for f := range fch {
			err := f.Fprint(os.Stdout)
			if err != nil {
				return err
			}
		}
		return wait()
	} else {
		var fs []*File
		for f := range fch {
			fs = append(fs, f)
		}
		err := FprintFiles(os.Stdout, fs...)
		if err != nil {
			return err
		}
		return wait()
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s:[Err]:%v\n", Name, err)
		os.Exit(1)
	}
}
