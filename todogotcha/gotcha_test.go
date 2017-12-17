package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestGather(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("t", "gather"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in      string
		exp     string
		wanterr bool
	}{
		{in: "hi", exp: ""},
		{in: "TODO: hi", exp: "L1:TODO: hi\n"},
		{in: "TODO: hello\nTODO: world\n", exp: "L1:TODO: hello\nL2:TODO: world\n"},
	}

	g := &gotcha{word: "TODO: "}
	buf := bytes.NewBufferString("")
	for i, test := range tests {
		t.Logf("[Case %d Start]", i)
		if err := ioutil.WriteFile(path, []byte(test.in), 0777); err != nil {
			t.Fatal(err)
		}
		if err := g.gatherWithBuffer(buf, path); err != nil {
			switch {
			case test.wanterr:
				t.Logf("[Wanterr]:%v", err)
			default:
				t.Error(err)
			}
			continue
		}
		if buf.String() != test.exp {
			t.Error("[Compare Error]")
			t.Errorf("[exp]:%s", test.exp)
			t.Errorf("[out]:%s", buf)
		} else {
			t.Logf("[Log]:%s", buf)
		}
		buf.Reset()
		t.Logf("[Case %d END]", i)
	}
}
