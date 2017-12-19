package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var TPath = func() string {
	TPath, err := filepath.Abs("t")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return TPath
}()

func TestMain(m *testing.M) {
	err := os.MkdirAll(filepath.Join(TPath, "d"), 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	defer os.Exit(code)
	os.RemoveAll(TPath)
}

func TestRun(t *testing.T) {
	err := ioutil.WriteFile(filepath.Join(TPath, "d", "hello.txt"), []byte("TODO: hello"), 0777)
	err = ioutil.WriteFile(filepath.Join(TPath, "hello.go"), []byte("// TODO: hello from go source"), 0777)
	if err != nil {
		t.Fatal(err)
	}
	// TODO: add cases
	tests := []struct {
		opt *option
		exp string
	}{
		{
			opt: &option{version: true},
			exp: name + " version " + version + "\n",
		},
		{
			opt: &option{root: filepath.Join(TPath, "d")},
			exp: string(filepath.Join(TPath, "d", "hello.txt")+"\n") +
				"L1:TODO: hello\n" +
				"\n",
		},
		{
			opt: &option{root: TPath, types: ".go"},
			exp: string(filepath.Join(TPath, "hello.go")+"\n") +
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
