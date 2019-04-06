package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

func TestWalk(t *testing.T) {
	dir := filepath.Join("testdata", "walker")
	w := NewWalker()
	resutQueue, _, err := w.Start("word", 0, dir)
	if err != nil {
		t.Fatal(err)
	}

	var out []*File
	for f := range resutQueue {
		out = append(out, f)
	}

	buf := bytes.NewBufferString("")
	err = FprintFiles(buf, out...)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("out:\n%v", buf)
}

var walerReadFileTests = []struct {
	in      string
	pat     string
	nlines  int
	exp     []*Context
	wanterr bool
}{
	{
		`word`,
		"word",
		0,
		[]*Context{
			{
				before: []*Line{},
				line:   &Line{1, "word"},
				after:  []*Line{},
			},
		},
		false,
	},
	{
		`word
hello
world
foo
bar
`,
		"world",
		1,
		[]*Context{
			{
				before: []*Line{{2, "hello"}},
				line:   &Line{3, "world"},
				after:  []*Line{{4, "foo"}},
			},
		},
		false,
	},
	{
		`word
hello world
word
foo
bar
`,
		"word",
		2,
		[]*Context{
			{
				before: []*Line{},
				line:   &Line{1, "word"},
				after:  []*Line{{2, "hello world"}},
			},
			{
				before: []*Line{},
				line:   &Line{3, "word"},
				after:  []*Line{{4, "foo"}, {5, "bar"}},
			},
		},
		false,
	},
	{
		`word
last one`,
		"word",
		2,
		[]*Context{
			{
				before: []*Line{},
				line:   &Line{1, "word"},
				after:  []*Line{{2, "last one"}},
			},
		},
		false,
	},
	{
		``,
		"word",
		0,
		nil,
		false,
	},
}

func TestWalkReadFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	verify := func(casev interface{}, exp []*Context, out []*Context) {
		t.Helper()
		if !reflect.DeepEqual(exp, out) {
			t.Logf("case %+v\nexp.cs:%+v\nout.cs:%+v", casev, exp, out)
			buf := bytes.NewBufferString("")
			for i, cs := range [][]*Context{exp, out} {
				var prefix string
				if i == 0 {
					prefix = "exp"
				} else {
					prefix = "oot"
				}
				FprintContexts(buf, "", cs)
				t.Logf("%s:\n%s", prefix, buf)
				buf.Reset()
			}
			t.FailNow()
		}
	}

	for _, test := range walerReadFileTests {
		f, err := ioutil.TempFile(tmpdir, "")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		_, err = f.WriteString(test.in)
		if err != nil {
			t.Fatal(err)
		}

		w := NewWalker()
		re, err := regexp.Compile(test.pat)
		if err != nil {
			t.Fatal(err)
		}
		w.regexp = re
		lq := new(LineQueue)
		out, err := w.readFile(f.Name(), lq, test.nlines)
		if test.wanterr {
			if err != nil {
				continue
			}
			t.Fatalf("want error but nil\ncase %+v", test)
		}
		if err != nil {
			t.Fatal(err)
		}
		verify(fmt.Sprintf("%+v", test), test.exp, out)
	}
}
