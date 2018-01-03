package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var testDir = func() string {
	TPath, err := filepath.Abs("t")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return TPath
}()

func TestMain(m *testing.M) {
	var code int
	defer func() { os.Exit(code) }()
	err := os.MkdirAll(filepath.Join(testDir), 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(testDir)
	code = m.Run()
}

// TODO: impl
func TestRun(t *testing.T) {
	root := filepath.Join(testDir, "run")
	if err := os.MkdirAll(root, 0777); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	newbufs := func() (buf *bytes.Buffer, errbuf *bytes.Buffer) {
		return bytes.NewBufferString(""), bytes.NewBufferString("")
	}
	opt.root = root
	newopt := func() *option {
		topt := *opt
		return &topt
	}

	// TODO: consider to split to subtests
	t.Run("valid exit", func(t *testing.T) {
		opt := newopt()
		root := filepath.Join(root, "valid_exit")
		if err := os.MkdirAll(root, 0777); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(filepath.Join(root, "hello.txt"), []byte("TODO: hello"), 0666); err != nil {
			t.Fatal(err)
		}
		buf, errbuf := newbufs()
		exp := filepath.Join(root, "hello.txt") + "\n" + "L1:TODO: hello" + "\n\n"

		testf := func() {
			if exit := run(buf, errbuf, opt); exit != ValidExit {
				t.Fatalf("exit=%d errbuf=%s", exit, errbuf)
			}
			if exp != buf.String() {
				t.Errorf("exp=%s\nout=%s", exp, buf)
			}
		}

		// valid exit
		testf()

		// use cache
		opt.cache = true
		buf.Reset()
		errbuf.Reset()
		testf()
		opt.cache = false

		// with total
		opt.total = true
		buf.Reset()
		errbuf.Reset()
		tmp := exp
		exp = exp + "files 1" + "\n" + "lines 1" + "\n"
		testf()
		exp = tmp
		opt.total = false

		// with sync
		opt.sync = true
		buf.Reset()
		errbuf.Reset()
		testf()
		opt.sync = false
	})

	t.Run("version", func(t *testing.T) {
		opt := newopt()
		buf, errbuf := newbufs()
		opt.version = true
		exit := run(buf, errbuf, opt)
		if exit != ValidExit {
			t.Error(errbuf)
			t.Error(buf)
		}
		exp := fmt.Sprintln(Name + " version " + Version)
		if buf.String() != exp {
			t.Errorf("exp=%s but out=%s err=%s", exp, buf, errbuf)
		}
	})

	t.Run("out", func(t *testing.T) {
		opt := newopt()
		buf, errbuf := newbufs()

		root := filepath.Join(root, "out")
		if err := os.MkdirAll(root, 0777); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(root)
		opt.root = root

		path := filepath.Join(root, "text.txt")
		contents := "TODO: hello"
		exp := path + "\n" + "L1:TODO: hello" + "\n\n"
		if err := ioutil.WriteFile(path, []byte(contents), 0666); err != nil {
			t.Fatal(err)
		}
		outpath := filepath.Join(root, "tmp.log")
		opt.out = outpath

		// out to tmp.log
		if exit := run(buf, errbuf, opt); exit != ValidExit {
			t.Fatalf("exit=%d errbuf=%s", exit, errbuf)
		}
		b, err := ioutil.ReadFile(outpath)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != exp {
			t.Errorf("exp=%s out=%s", exp, buf)
		}

		// reject override
		buf.Reset()
		errbuf.Reset()
		if exit := run(buf, errbuf, opt); exit != ErrInitialize {
			t.Fatalf("[reject override] expected exit=%d exit=%d errbuf=%s opt=%#v", ErrInitialize, exit, errbuf, opt)
		}

		// use force
		opt.force = true
		buf.Reset()
		errbuf.Reset()
		if exit := run(buf, errbuf, opt); exit != ValidExit {
			t.Fatalf("[use force] exit=%d errbuf=%s opt=%#v", exit, errbuf, opt)
		}

		// reject directory
		buf.Reset()
		errbuf.Reset()
		dir := filepath.Join(root, "dir")
		if err := os.Mkdir(dir, 0777); err != nil {
			t.Fatal(err)
		}
		opt.out = dir
		opt.force = true
		if exit := run(buf, errbuf, opt); exit != ErrInitialize {
			t.Fatalf("[reject directory] expected exit=%d exit=%d errbuf=%s opt=%#v", ErrInitialize, exit, errbuf, opt)
		}
	})

	t.Run("verbose", func(t *testing.T) {
		root := filepath.Join(root, "verbose")
		if err := os.Mkdir(root, 0777); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "toolong.txt")
		if err := ioutil.WriteFile(path, []byte(TooLongLine), 0666); err != nil {
			t.Fatal(err)
		}
		opt := newopt()
		opt.verbose = true
		buf, errbuf := newbufs()
		if exit := run(buf, errbuf, opt); exit == ValidExit {
			t.Fatal("expected error but valid exit")
		}
	})

	t.Run("specify file", func(t *testing.T) {
		root := filepath.Join(root, "specify_file")
		if err := os.Mkdir(root, 0777); err != nil {
			t.Fatal(err)
		}
		opt := newopt()
		buf, errbuf := newbufs()
		path := filepath.Join(root, "specify.txt")
		if err := ioutil.WriteFile(path, []byte("TODO: hello"), 0666); err != nil {
			t.Fatal(err)
		}
		exp := path + "\n" + "L1:TODO: hello" + "\n\n"
		opt.root = path
		if exit := run(buf, errbuf, opt); exit != ValidExit {
			t.Errorf("exit=%d errbuf=%s opt=%#v", exit, errbuf, opt)
		}
		if exp != buf.String() {
			t.Errorf("exp=%s out=%s opt=%s", exp, buf, opt)
		}
	})
}
