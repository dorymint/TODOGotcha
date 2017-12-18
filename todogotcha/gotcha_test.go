package main

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGather(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("t", "gather"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in      string
		exp     []string
		wanterr bool
	}{
		{in: "hi", exp: nil},
		{in: "TODO: hi", exp: []string{"L1:TODO: hi"}},
		{in: "TODO: hello\nTODO: world\n", exp: []string{"L1:TODO: hello", "L2:TODO: world"}},
	}

	g := NewGotcha()
	for i, test := range tests {
		t.Logf("[Case %d Start]", i)
		if err := ioutil.WriteFile(path, []byte(test.in), 0777); err != nil {
			t.Fatal(err)
		}
		res := g.gather(path)
		if res.err != nil {
			switch {
			case test.wanterr:
				t.Logf("[Log err]:%#v", res.err)
			default:
				t.Errorf("[Unexpected error]:%#v", res.err)
			}
			continue
		}
		if reflect.DeepEqual(res.contents, test.exp) {
			t.Logf("[Log res]:%#v", res)
		} else {
			t.Error("[Compare Error]")
			t.Errorf("[exp]:%#v", test.exp)
			t.Errorf("[out]:%#v", res.contents)
		}
		t.Logf("[Case %d END]", i)
	}
}
