package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tpath, err := filepath.Abs("t")
	if err != nil {
		t.Fatal(err)
	}

	// TODO: add cases
	tests := []struct {
		opt *option
		exp string
	}{
		{
			opt: &option{root: tpath},
			exp: filepath.Join(tpath, "hello.txt") + "\n" + "L1:TODO: hello" + "\n",
		},
	}

	for i, test := range tests {
		prefix := fmt.Sprintf("CASE [%d]:", i)
		buf := bytes.NewBufferString("")
		errbuf := bytes.NewBufferString("")
		exitCode := run(buf, errbuf, test.opt)
		switch {
		case exitCode != ValidExit:
			t.Error("Start:", prefix)
			t.Error("exitCode:", exitCode)
			t.Error("exp:", test.exp)
			t.Error("buf:", buf)
			t.Error("errbuf:", errbuf)
			t.Error("End:", prefix)
		case strings.Compare(test.exp, buf.String()) == 0:
			t.Error("Start:", prefix)
			t.Error("exp:", test.exp)
			t.Error("out:", buf)
			t.Error("End:", prefix)
		default:
			t.Log("Start:", prefix)
			t.Log("Log:", "buf:", buf)
			t.Log("Log:", "errbuf:", errbuf)
			t.Log("End:", prefix)
		}
	}
}
