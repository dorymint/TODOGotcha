package main


// TODO: Modified flgas check!!!



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

// filemap is map[dirname][]files
func makeTempDirs() (tmproot string) {
	tmproot, err := ioutil.TempDir("", "crawl")
	if err != nil {
		log.Fatal(err)
	}

	// create tempdirs
	for _, x := range TmpDirs {
		dirpath := filepath.Join(tmproot, x)
		if err := os.MkdirAll(dirpath, 0700); err != nil {
			log.Fatal(err)
		}

		// create tempfiles
		for _, y := range TmpFilesMap[x] {
			filepath := filepath.Join(dirpath, y)
			if err := ioutil.WriteFile(filepath, nil, 0600); err != nil {
				log.Fatal(err)
			}
		}
	}
	return tmproot
}

func TestMain(m *testing.M) {
	result := func() int {
		// make temp
		TmpRoot = makeTempDirs()
		log.Printf("tmproot=%v\n", TmpRoot)
		defer func() {
			if err := os.RemoveAll(TmpRoot); err != nil {
				log.Fatal(err)
			}
			log.Println("tmproot deleted")
		}()

		// TmpRoot + TmpDirs
		for i, x := range TmpDirs {
			TmpDirs[i] = filepath.Join(TmpRoot, x)
		}
		return m.Run()
	}()
	os.Exit(result)
}

// TODO: To simple! delete this?
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

// TODO: To simple!!
func TestDrisCrawl(t *testing.T) {
	// Create expected os.FileInfo Map
	expectedInfosMap := make(map[string][]os.FileInfo)
	for _, dirname := range append(TmpDirs, TmpRoot) {
		func() {
			f, err := os.Open(dirname)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if errclose := f.Close(); errclose != nil {
					t.Fatal(errclose)
				}
			}()
			infos, err := f.Readdir(0)
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
		_ = dirsCrawl(*flags.root)
	}
}

func TestSuffixSearcher(t *testing.T) {
	targetSuffix := []string{
		"go",
		"txt",
	}
	fileNames := []string{
		"test1.go",
		"test2.txt",
	}
	ignoreNames := []string{
		"fata1go",
		"fatal2txt",
	}

	for _, x := range fileNames {
		if !suffixSearcher(x, targetSuffix) {
			t.Fatalf("expected return true, but false %v\n", x)
		}
	}
	for _, x := range ignoreNames {
		if suffixSearcher(x, targetSuffix) {
			t.Fatalf("expected return false, but true %v\n", x)
		}
	}
}

func writeContent(t *testing.T, content string) string {
	f, err := ioutil.TempFile("", "todogotcha")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if errclose := f.Close(); errclose != nil {
			t.Fatal(errclose)
		}
	}()

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestGather(t *testing.T) {
	type test struct {
		filecontent string
		expected    []string
		keyword     string
	}
	tests := []test{

		// patern1
		{
			filecontent: `
in the file content
// TODO: Test
end
`,
			expected: []string{
				"L3:" + " Test",
			},
			keyword: "TODO:",
		},

		// patern2
		{
			filecontent: `
case non target retunr nil
`,
			expected: nil,
			keyword:  "TODO:",
		},

		// patern3
		{
			filecontent: `
case 2line and many keywords
TODO: TODO: TODO:
TODO: 2line
`,
			expected: []string{
				"L3:" + " TODO: TODO:",
				"L4:" + " 2line",
			},
			keyword: "TODO:",
		},

		// patern4
		{
			filecontent: `
case changed keyword
// NEXT: Test
end
`,
			expected: []string{
				"L3:" + " Test",
			},
			keyword: "NEXT:",
		},

		// patern5
		{
			filecontent: `
case empty keyword
`,
			expected: []string{
				"L1:" + "",
				"L2:" + "case empty keyword",
			},
			keyword: "",
		},
		// TODO: add test case
	}

	run := func(data test, keyword string) {
		filename := writeContent(t, data.filecontent)
		defer func() {
			if err := os.Remove(filename); err != nil {
				t.Fatal(err)
			}
		}()
		out := gather(filename, keyword)
		if !reflect.DeepEqual(data.expected, out) {
			t.Error("not equal!")
			t.Error("expected")
			for _, x := range data.expected {
				t.Error(x)
			}
			t.Error("but out")
			for _, x := range out {
				t.Error(x)
			}
			t.FailNow()
		}
	}
	// Test
	for _, x := range tests {
		run(x, x.keyword)
	}
}

func deepEqualMaps(t *testing.T, expected, out map[string][]string) {
	if !reflect.DeepEqual(expected, out) {
		t.Error("not uqual!!")
		t.Error("expected")
		for key, x := range expected {
			t.Error(key, x)
		}
		t.Error("but out")
		for key, x := range out {
			t.Error(key, x)
		}
		t.FailNow()
	}
}

// TODO: Create test data and run
func TestUnlimitedGopherWorks(t *testing.T) {
	// empty paturn
	// TODO: Add another case
	// pre request is dirsCrawl Green!
	infomap := dirsCrawl(TmpRoot)

	expected := make(map[string][]string)
	out := unlimitedGopherWorks(infomap, []string{"go", "txt"}, "TODO:")
	deepEqualMaps(t, expected, out)
}

// Integration test
func TestGophersProc(t *testing.T) {
	// empty
	// TODO: Add another case
	out := GophersProc(TmpRoot)
	expected := make(map[string][]string)
	deepEqualMaps(t, expected, out)
}

// TODO: Add another case
func ExampleOutputTODOList() {
	todoMap := make(map[string][]string)
	todoMap["/tmp/test"] = []string{"L1: test"}

	// flgas
	*flags.sort = "off"
	*flags.result = "off"
	OutputTODOList(todoMap)
	// Unordered Output:
	// /tmp/test
	// L1: test
}
func ExampleOutputTODOList_sortON() {
	todoMap := make(map[string][]string)
	todoMap["/tmp/test"] = []string{"L1: test"}
	todoMap["/a"] = []string{"L2: sort on"}

	// flgas
	*flags.sort = "on"
	*flags.result = "off"
	OutputTODOList(todoMap)
	// Output:
	// /a
	// L2: sort on
	//
	// /tmp/test
	// L1: test
}
