package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
)

// TODO: fix
func TestWalk(t *testing.T) {
	dir := filepath.Join("testdata", "walker")

	w := NewWalker()
	var err error
	err = w.SetRegexp("word")
	if err != nil {
		t.Fatal(err)
	}
	rec, wait := w.Start()
	err = w.SendPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	go wait()

	buf := bytes.NewBufferString("")
	for f := range rec {
		buf.WriteString(fmt.Sprintln(f.Path))
		for _, c := range f.Contexts {
			buf.WriteString(fmt.Sprintln(c.String()))
		}
		buf.WriteString(fmt.Sprintln())
	}
	t.Logf("out:\n%v", buf)
}
