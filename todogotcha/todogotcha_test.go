package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// testing directory structure
var (
	TmpRoot = ""
	TmpDirs = []string{
		"A",
		"A/aA",
		"A/aB",
		"B",
		"B/bA",
		"B/bA/baA",
	}
	TmpFilesMap = map[string][]string{
		TmpDirs[0]: {
			"A.txt",
		},
		TmpDirs[1]: {
			"AA.1.txt",
			"AA.2.txt",
		},
		TmpDirs[2]: {
			"AAA.1.txt",
			"AAA.2.txt",
			"AAA.3.txt",
		},
		TmpDirs[3]: {
			"B.go",
		},
	}
)

func TestMain(m *testing.M) {
	result := func() int {
		// flag
		var err error

		// TODO:for the moment
		*root, err = filepath.Abs("../../")
		if err != nil {
			log.Fatalf("TestMain:%v\n", err)
		}

		// make temp
		TmpRoot = makeTempDir()
		defer func() {
			if err := os.RemoveAll(TmpRoot); err != nil {
				log.Fatal(err)
			}
			log.Println("tmproot deleted")
		}()
		log.Printf("tmproot is = %v\n", TmpRoot)

		// define dirspath string(tmpRoot + <tmpDirs>)
		for i, x := range TmpDirs {
			TmpDirs[i] = filepath.Join(TmpRoot, x)
		}
		return m.Run()
	}()
	os.Exit(result)
}

// filemap is map[dirname][]files
func makeTempDir() (root string) {
	tmproot, err := ioutil.TempDir("", "crawl")
	if err != nil {
		log.Fatal(err)
	}

	// create tempdir
	for _, x := range TmpDirs {
		dirpath := filepath.Join(tmproot, x)
		if err := os.MkdirAll(dirpath, 0700); err != nil {
			log.Fatal(err)
		}

		// create tempfile
		for _, y := range TmpFilesMap[x] {
			filepath := filepath.Join(dirpath, y)
			if err := ioutil.WriteFile(filepath, nil, 0700); err != nil {
				log.Fatal(err)
			}
		}
	}
	return tmproot
}

func deepEqualStrings(t *testing.T, expected, out []string) {
	if reflect.DeepEqual(expected, out) {
		return
	}
	t.Error("Not Equal!")
	t.Error("expected")
	for _, x := range expected {
		t.Error(x)
	}
	t.Error("but out")
	for _, x := range out {
		t.Error(x)
	}
	t.FailNow()
}

func TestDrisCrawl(t *testing.T) {
	// Create expected os.FileInfo Map
	expectedInfosMap := make(map[string][]os.FileInfo)
	for _, dirname := range append(TmpDirs, TmpRoot) {
		func() {
			f, err := os.Open(dirname)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			infos, err  := f.Readdir(0)
			if err != nil {
				t.Fatal(err)
			}
			expectedInfosMap[dirname] = infos
		}()
	}

	// Run
	outInfosMap := dirsCrawl(TmpRoot)

	var expected []string
	for _, infos := range expectedInfosMap {
		for _, info := range infos {
			expected = append(expected, info.Name())
		}
	}
	var out []string
	for _, infos := range outInfosMap {
		for _, info := range infos {
			out = append(out, info.Name())
		}
	}
	sort.Strings(expected)
	sort.Strings(out)
	deepEqualStrings(t, expected, out)
}

func BenchmarkDirsCrawl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dirsCrawl(*root)
	}
}

func TestSuffixSearcher(t *testing.T) {
	targetSuffix := []string{
		"go",
		"txt",
	}
	filename := []string{
		"test1.go",
		"test2.txt",
	}
	fatalname := []string{
		"fata1go",
		"fatal2txt",
	}

	for _, x := range filename {
		if !suffixSeacher(x, targetSuffix) {
			t.Fatalf("expected true, but false %v\n", x)
		}
	}
	for _, x := range fatalname {
		if suffixSeacher(x, targetSuffix) {
			t.Fatalf("expected false, but true %v\n", x)
		}
	}
}

func writeContent(t *testing.T, content string) string {
	f, err := ioutil.TempFile("", "level2Test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// TODO:Create test data tuple list
func TestGather(t *testing.T) {
	// input content
	filename := writeContent(t, `// TODO:Test`)
	defer func() {
		if err := os.Remove(filename); err != nil {
			t.Fatal(err)
		}
	}()

	searchWord := "TODO:"
	expected := []string{
		"L1:" + "Test",
	}

	out, err := gather(filename, searchWord)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, out) {
		t.Error("not equal!")
		t.Error("expected")
		for _, x := range expected {
			t.Error(x)
		}
		t.Error("but out")
		for _, x := range out {
			t.Error(x)
		}
		t.FailNow()
	}
}
