package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const TooLongLine = `too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line too long line`

func Test_gather(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("t", "gather.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	type Tests struct {
		in  string
		exp *gatherRes
	}

	verify := func(t *testing.T, g *Gotcha, tests []Tests) {
		for _, test := range tests {
			f.Truncate(0)
			f.Seek(0, 0)
			f.WriteString(test.in)
			res := g.gather(path)
			if !reflect.DeepEqual(test.exp, res) {
				t.Errorf("not equel: exp=%#v out=%#v", test.exp, res)
			}
		}
	}

	t.Run("none error", func(t *testing.T) {
		g := NewGotcha()
		tests := []Tests{
			{
				in: "TODO: hi",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:TODO: hi"},
					err:      nil,
				},
			},
			{
				in: "TODO: hello\nTODO: world\n",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:TODO: hello", "L2:TODO: world"},
					err:      nil,
				},
			},
		}
		verify(t, g, tests)
	})

	t.Run("use trim", func(t *testing.T) {
		g := NewGotcha()
		g.Trim = true
		tests := []Tests{
			{
				in: "TODO: hi",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:hi"},
					err:      nil,
				},
			},
			{
				in: "TODO: hello\nTODO: world\n",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:hello", "L2:world"},
					err:      nil,
				},
			},
		}
		verify(t, g, tests)
	})

	t.Run("use add", func(t *testing.T) {
		g := NewGotcha()
		g.Add = 1
		tests := []Tests{
			{
				in: "TODO: hi\nnext 1 line\nnext 2 line",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:TODO: hi", " 2:next 1 line"},
					err:      nil,
				},
			},
			{
				in: "TODO: hello\nTODO: world\n",
				exp: &gatherRes{
					path:     path,
					contents: []string{"L1:TODO: hello", "L2:TODO: world"},
					err:      nil,
				},
			},
		}
		verify(t, g, tests)
	})

	/// expected return errors

	t.Run("err file not exists", func(t *testing.T) {
		notexists := path + "notexits"
		g := NewGotcha()
		res := g.gather(notexists)
		if os.IsNotExist(res.err) {
			return
		}
		t.Errorf("expected error is %#v but out %#v", os.ErrNotExist, res.err)
	})

	t.Run("err have too long line", func(t *testing.T) {
		g := NewGotcha()
		tests := []Tests{
			{
				in: TooLongLine,
				exp: &gatherRes{
					path:     path,
					contents: nil,
					err:      ErrHaveTooLongLine,
				},
			},
		}
		verify(t, g, tests)
	})
}

func Test_gatherResErr(t *testing.T) {
	tests := []struct {
		gr  *gatherRes
		exp error
	}{
		{
			gr: &gatherRes{
				path:     "path",
				contents: nil,
				err:      ErrHaveTooLongLine,
			},
			exp: &gatherRes{
				path:     "path",
				contents: nil,
				err:      ErrHaveTooLongLine,
			},
		},
		{
			gr: &gatherRes{
				path:     "path",
				contents: nil,
				err:      nil,
			},
			exp: nil,
		},
		{
			gr: &gatherRes{
				path:     "path",
				contents: nil,
				err:      os.ErrPermission,
			},
			exp: os.ErrPermission,
		},
	}

	for _, test := range tests {
		err := test.gr.Err()
		if !reflect.DeepEqual(test.exp, err) {
			t.Errorf("exp=%#v but out=%#v", test.exp, err)
		}
		if err != nil {
			t.Logf("[Log] gr.Error(): %#v", test.gr.Error())
		}
	}
}

func TestFwrite(t *testing.T) {
	buf := bytes.NewBufferString("")
	tests := []struct {
		gr      *gatherRes
		exp     string
		wanterr bool
	}{
		{
			gr: &gatherRes{
				path:     "path",
				contents: []string{"hi"},
				err:      nil,
			},
			exp:     "path\n" + "hi\n\n",
			wanterr: false,
		},
		{
			gr: &gatherRes{
				path:     "path",
				contents: nil,
				err:      nil,
			},
			exp:     "",
			wanterr: false,
		},
		{
			gr: &gatherRes{
				path:     "path",
				contents: nil,
				err:      ErrHaveTooLongLine,
			},
			exp:     "",
			wanterr: true,
		},
	}

	for _, test := range tests {
		buf.Reset()
		err := test.gr.Fwrite(buf)
		if test.wanterr && err == nil {
			t.Error("expected error but nil")
		}
		if test.exp != buf.String() {
			t.Errorf("exp=%#v but out=%#v", test.exp, buf.String())
		}
	}
}

func Test_isTooLong(t *testing.T) {
	tests := []struct {
		in       error
		wantbool bool
	}{
		// want true
		{
			in:       ErrHaveTooLongLine,
			wantbool: true,
		},
		{
			in: &gatherRes{
				path:     "gatherRes",
				contents: nil,
				err:      ErrHaveTooLongLine,
			},
			wantbool: true,
		},

		// want false
		{
			in:       nil,
			wantbool: false,
		},
		{
			in:       errors.New("new error"),
			wantbool: false,
		},
		{
			in: &gatherRes{
				path:     "wantfalse",
				contents: nil,
				err:      nil,
			},
			wantbool: false,
		},
	}
	for _, test := range tests {
		b := IsTooLong(test.in)
		if test.wantbool != b {
			t.Errorf("unexpected: want=%#v out=%#v", test.wantbool, b)
		}
	}
}

func Test_isTarget(t *testing.T) {
	type Tests struct {
		path   string
		target bool
	}
	f := func(t *testing.T, g *Gotcha, tests []Tests) {
		for _, test := range tests {
			b := g.isTarget(test.path)
			if test.target != b {
				t.Errorf("path=%#v want bool is %#v but %#v", test.path, test.target, b)
			}
		}
	}
	t.Run("use ignore base", func(t *testing.T) {
		g := NewGotcha()
		g.IgnoreBasesMap["ignore"] = true
		tests := []Tests{
			{
				path:   "ignore",
				target: false,
			},
			{
				path:   "target.txt",
				target: true,
			},
		}
		f(t, g, tests)
	})

	t.Run("ignore type", func(t *testing.T) {
		g := NewGotcha()
		tests := []Tests{
			{
				path:   "ignore.bz",
				target: false,
			},
			{
				path:   "target.txt",
				target: true,
			},
		}
		f(t, g, tests)
	})
}

// TODO: impl
func TestAsyncWorkGo(t *testing.T) {
	root := filepath.Join(testDir, "async_work_go")
	if err := os.MkdirAll(root, 0777); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	t.Run("valid exit", func(t *testing.T) {
		r := filepath.Join(root, "valid_exit")
		if err := os.MkdirAll(r, 0777); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(r)

		path := filepath.Join(r, "hello.txt")
		contents := "// TODO: hello"

		if err := ioutil.WriteFile(path, []byte(contents), 0666); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		exp := path + "\n" + "L1:" + contents + "\n\n"
		buf := bytes.NewBufferString("")
		errbuf := bytes.NewBufferString("")
		g := NewGotcha()
		g.W = buf
		g.Log.SetOutput(errbuf)

		exitCode := g.WorkGo(root, 0)
		if exitCode != ValidExit {
			t.Fatalf("expected valid exit but return %d: errbuf:%s", exitCode, errbuf.String())
		}
		if exp != buf.String() {
			t.Fatalf("exp:%s but out:%s", exp, buf.String())
		}
	})

	t.Run("expected error", func(t *testing.T) {
		r := filepath.Join(root, "expected_error")
		if err := os.MkdirAll(r, 0777); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(r)

		path := filepath.Join(r, "toolong.txt")
		contents := TooLongLine

		if err := ioutil.WriteFile(path, []byte(contents), 0666); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(path)

		errbuf := bytes.NewBufferString("")
		g := NewGotcha()
		g.Log.SetOutput(errbuf)

		out := g.WorkGo(root, 0)
		if out == 0 {
			t.Fatalf("expected !0 but out 0: errbuf:%s", errbuf.String())
		}
	})
}

func TestSyncWorkGo(t *testing.T) {
	// make root
	root := filepath.Join(testDir, "sync_work_go")
	if err := os.MkdirAll(root, 0777); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	// make directory
	dir := filepath.Join(root, "dir")
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatal(err)
	}

	// make file
	file := filepath.Join(dir, "hello.txt")
	if err := ioutil.WriteFile(file, []byte("TODO: hello"), 0666); err != nil {
		t.Fatal(err)
	}

	t.Run("gotcha", func(t *testing.T) {
		g := NewGotcha()
		buf := bytes.NewBufferString("")
		g.W = buf
		if err := g.SyncWorkGo(root); err != nil {
			t.Fatal(err)
		}
		exp := file + "\n" + "L1:TODO: hello\n\n"
		if buf.String() != exp {
			t.Fatalf("exp=%#v but out=%#v", exp, buf.String())
		}
	})

	t.Run("ignore dir", func(t *testing.T) {
		g := NewGotcha()
		buf := bytes.NewBufferString("")
		g.W = buf
		g.IgnoreDirsMap[filepath.Base(dir)] = true
		if err := g.SyncWorkGo(root); err != nil {
			t.Fatal(err)
		}
		exp := ""
		if buf.String() != exp {
			t.Fatalf("exp=%#v but out=%#v", exp, buf.String())
		}
	})

	// TODO: consider case of g.Log.Fatal
	t.Run("valid err", func(t *testing.T) {
		g := NewGotcha()
		errbuf := bytes.NewBufferString("")
		g.W = ioutil.Discard
		g.Log.SetOutput(errbuf)

		tooLongFile := filepath.Join(root, "too_long")
		ioutil.WriteFile(tooLongFile, []byte(TooLongLine), 0666)
		if err := g.SyncWorkGo(root); err != nil {
			t.Fatal(err)
		} else {
			t.Log(errbuf)
		}
	})
}
