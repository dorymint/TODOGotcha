package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var tpath = func() string {
	tpath, err := filepath.Abs("t")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return tpath
}()

func TestRun(t *testing.T) {
	// TODO: add cases
	tests := []struct {
		opt *option
		exp string
	}{
		{
			opt: &option{version: true},
			exp: "gotcha version " + version + "\n",
		},
		{
			opt: &option{root: filepath.Join(tpath, "d")},
			exp: string(filepath.Join(tpath, "d", "hello.txt")+"\n") +
				"L1:TODO: hello\n" +
				"\n",
		},
		{
			opt: &option{root: tpath, types: ".go"},
			exp: string(filepath.Join(tpath, "hello.go")+"\n") +
				"L1:// TODO: hello from go source\n" +
				"\n",
		},
	}

	for i, test := range tests {
		prefix := fmt.Sprintf("CASE [%d]:", i)
		buf := bytes.NewBufferString("")
		errbuf := bytes.NewBufferString("")
		exitCode := run(buf, errbuf, test.opt)
		switch {
		case exitCode != ValidExit:
			t.Error("Fail Start:", prefix)
			t.Error("exitCode:", exitCode)
			t.Error("exp:", test.exp)
			t.Error("buf:", buf)
			t.Error("errbuf:", errbuf)
			t.Error("End:", prefix)
		case test.exp != buf.String():
			t.Error("Fail Start:", prefix)
			t.Error("exp:", test.exp)
			t.Error("out:", buf)
			t.Error("End:", prefix)
		default:
			t.Log("Pass Start:", prefix)
			t.Log("Log:", "buf:", buf)
			t.Log("Log:", "errbuf:", errbuf)
			t.Log("End:", prefix)
		}
	}
}
