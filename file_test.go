package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

var readFileTests = []struct {
	str             string
	pat             string
	nbefore, nafter int

	exp string
}{
	{
		str: "hello world",
		pat: "world",
		exp: "1:hello world",
	},
}

func TestReadFile(t *testing.T) {
	tmp, err := ioutil.TempDir("", "test_readfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	tmpf, err := ioutil.TempFile(tmp, "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpf.Name())
	defer tmpf.Close()
	reset := func() {
		err = tmpf.Truncate(0)
		if err == nil {
			_, err = tmpf.Seek(0, 0)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	scontexts := func(cs []*Context) (str string) {
		for _, c := range cs {
			str += fmt.Sprint(c.String())
		}
		return str
	}

	// TODO: compare
	for _, test := range readFileTests {
		_, err = tmpf.WriteString(test.str)
		if err != nil {
			t.Fatal(err)
		}
		fr := NewFileReader(regexp.MustCompile(test.pat), test.nbefore, test.nafter)
		out, err := fr.ReadFile(tmpf.Name())
		if err != nil {
			t.Fatal(err)
		}
		fr.Reset()
		t.Logf("out:%+v\nPath:%s\nContexts:%s\n", out, out.Path, scontexts(out.Contexts))
		reset()
	}
}
