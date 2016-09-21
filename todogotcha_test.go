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
	tmpRoot = ""
	tmpDirs = []string{
		"A",
		"A/aA",
		"A/aB",
		"B",
		"B/bA",
		"B/bA/baA",
	}
	tmpFilesMap = map[string][]string{
		tmpDirs[0]: {
			"A.txt",
		},
		tmpDirs[1]: {
			"AA.1.txt",
			"AA.2.txt",
		},
		tmpDirs[2]: {
			"AAA.1.txt",
			"AAA.2.txt",
			"AAA.3.txt",
		},
		tmpDirs[3]: {
			"B.go",
		},
	}
)

func TestMain(m *testing.M) {
	// flag
	var err error
	*root, err = filepath.Abs("../../")
	if err != nil {
		log.Fatalf("TestMain:%v\n", err)
	}

	// make temp
	tmpRoot = makeTempDir(tmpDirs, tmpFilesMap)
	defer func() {
		if err := os.RemoveAll(tmpRoot); err != nil {
			log.Fatal(err)
		}
	}()
	log.Printf("tmproot is = %v\n", tmpRoot)

	os.Exit(m.Run())
}

// filemap is map[dirname][]files
func makeTempDir(dirs []string, filemap map[string][]string) (root string) {

	tmproot, err := ioutil.TempDir("", "crawl")
	if err != nil {
		log.Fatal(err)
	}

	// create tempdir
	for _, x := range dirs {
		dirpath := filepath.Join(tmproot, x)
		if err := os.MkdirAll(dirpath, 0700); err != nil {
			log.Fatal(err)
		}

		// create tempfile
		for _, y := range filemap[x] {
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

// TODO:読みづらい、何とかしたい
func TestDrisCrawl(t *testing.T) {
	var expectedDirs []string
	for _, x := range tmpDirs {
		expectedDirs = append(expectedDirs, filepath.Join(tmpRoot, x))
	}

	// Run
	outDirs, outInfosMap := dirsCrawl(tmpRoot)

	// dirs check
	sort.Strings(outDirs)
	deepEqualStrings(t, expectedDirs, outDirs)

	// Create expected os.FileInfo Map
	expectedFileInfosMap := make(map[string][]os.FileInfo)
	for _, dirname := range append(expectedDirs, tmpRoot) {
		f, err := os.Open(dirname)
		if err != nil {
			t.Fatal(err)
		}
		info, err := f.Readdir(0)
		if err != nil {
			t.Fatal(err)
		}
		expectedFileInfosMap[dirname] = info
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// CreateTestData expected info names
	var expectedNames []string
	for _, infos := range expectedFileInfosMap {
		for _, info := range infos {
			expectedNames = append(expectedNames, info.Name())
		}
	}
	sort.Strings(expectedNames)
	// CreateTestData out info names
	var outNames []string
	for _, infos := range outInfosMap {
		for _, info := range infos {
			outNames = append(outNames, info.Name())
		}
	}
	// Check []os.FileInfo names
	sort.Strings(outNames)
	deepEqualStrings(t, expectedNames, outNames)
}

func BenchmarkUseWaitGroup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dirsCrawl(*root)
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
	filename := writeContent(t, `// TODO:Test`)
	defer func() {
		if err := os.Remove(filename); err != nil {
			t.Fatal(err)
		}
	}()

	expected := []string{
		"L1:" + "TODO:Test",
	}

	// TEST!
	out, err := gather(filename, "TODO")
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


