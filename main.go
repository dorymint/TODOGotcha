// rgr is cli tool for find target words in many files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	Version = "0.2.0dev"
	Name    = "rgr"
)

var opt struct {
	help    bool
	version bool

	root string
	word string

	// TODO
	trim bool

	// TODO
	// append lines
	before uint
	after  uint

	verbose bool
}

// TODO
func init() {
	flag.BoolVar(&opt.help, "help", false, "Print usage")
	flag.BoolVar(&opt.version, "version", false, "Print version")

	flag.StringVar(&opt.root, "root", "", "Specify search root directory")
	flag.StringVar(&opt.word, "word", "", "Specify target words")

	flag.BoolVar(&opt.verbose, "verbose", false, "Verbose output")
}

func run() error {
	flag.Parse()
	switch flag.NArg() {
	case 0:
		// pass
	case 1:
		// TODO: consider
	default:
		// TODO: consider
	}

	// version
	switch {
	case opt.help:
		// TODO
		return nil
	case opt.version:
		_, err := fmt.Printf("%s %s\n", Name, Version)
		return err
	}

	// TODO
	return errors.New("not implemented")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
