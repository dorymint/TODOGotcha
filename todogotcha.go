package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TODO:LIST
// parse flags
// search dirs
// search in files
// show result

// flags
var (
	root       = flag.String("root", "./", "Specify search root")
	suffix     = flag.String("filetype", "go txt", `Specify target file type into the " "`)
	suffixList []string
	gatherTarget = flag.String("key", "TODO", `Specify gather target keyword`)
)

func init() {
	var err error
	flag.Parse()
	*root, err = filepath.Abs(*root)
	if err != nil {
		log.Fatalf("init:%v\n", err)
	}
	suffixList = strings.Split(*suffix, " ")
	argsCheck()
}

// Checking after parsing flags
func argsCheck() {
	if len(flag.Args()) != 0 {
		fmt.Printf("cmd = %v\n\n", os.Args)
		fmt.Printf("-----| Unknown option |-----\n\n")
		for _, x := range flag.Args() {
			fmt.Println(x)
		}
		fmt.Printf("\n")
		fmt.Println("-----| Usage |-----")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

// Use wait group!!
// もう少しシンプルにしたい
func dirsCrawl(root string) ([]string, map[string][]os.FileInfo) {

	// mux group
	var Dirlist []string
	DirsCache := make(map[string]bool)
	InfoCache := make(map[string][]os.FileInfo)
	Mux := new(sync.Mutex)

	Wg := new(sync.WaitGroup)

	var crawl func(string)
	crawl = func(dirname string) {
		defer Wg.Done()

		Mux.Lock()
		if DirsCache[dirname] {
			return
		}
		DirsCache[dirname] = true
		Mux.Unlock()

		f, err := os.Open(dirname)
		if err != nil {
			log.Printf("crawl:%v\n", err)
			return
		}
		defer func() {
			if errclose := f.Close(); errclose != nil {
				log.Println(errclose)
			}
		}()

		info, err := f.Readdir(0)
		if err != nil {
			log.Printf("crawl info:%v", err)
			return
		}
		Mux.Lock()
		InfoCache[dirname] = info
		Mux.Unlock()

		for _, x := range info {
			if x.IsDir() {
				tmp := filepath.Join(dirname, x.Name())
				Mux.Lock()
				Dirlist = append(Dirlist, tmp)
				Mux.Unlock()
				Wg.Add(1)
				go crawl(tmp)
			}
		}
	}

	Wg.Add(1)
	crawl(root)
	Wg.Wait()
	return Dirlist, InfoCache
}

// TODO:Do erase after test
func crawlshow() {
	dirslist, infomap := dirsCrawl(*root)
	fmt.Println("show directory list")
	for _, x := range dirslist {
		fmt.Println(x)
	}
	fmt.Println("root = ", *root)
	fmt.Println("length", len(dirslist))

	fmt.Println("show target files")
	var fileCount int
	for dirname, infos := range infomap {
		for _, info := range infos {
			if suffixSeacher(info.Name(), suffixList) {
				fmt.Println(filepath.Join(dirname, info.Name()))
				fileCount++
				time.Sleep(time.Millisecond)
			}
		}
	}
	fmt.Printf("all dirs = %v\nfile count = %v\n", len(dirslist), fileCount)
}

// suffixList
func suffixSeacher(filename string, targetSuffix []string) bool {
	for _, x := range targetSuffix {
		if strings.HasSuffix(filename, "."+x) {
			return true
		}
	}
	return false
}

// specify filename and target, Gather target(TODOs), return todoList.
// TODO:use goroutine
func gather(filename string, target string) ([]string, error) {
	var todoList []string
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("todoGather:%v\n", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("todoGather:%v", err)
		}
	}()

	for sc, i := bufio.NewScanner(f), uint(1); sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("todoGather:%v\n", err)
		}
		if index := strings.Index(sc.Text(), target); index != -1 {
			todoList = append(todoList, fmt.Sprintf("L%v:%s", i, sc.Text()[index:]))
		}
	}
	return todoList, nil
}



// TODOGotcha!! main proc
// TODO:use goroutine
// TODO:驚くべき読みにくさ...何とかしたい
// 多分データ構造の選択を間違えてる
func mainproc() (todoMap map[string][]string, gatherErr error) {
	todoMap = make(map[string][]string)
	_, infomap := dirsCrawl(*root)
	for dirname, infos := range infomap {
		for _, info := range infos {
			if suffixSeacher(info.Name(), suffixList) {
				tmp, err := gather(filepath.Join(dirname, info.Name()), *gatherTarget)
				if err != nil {
					log.Printf("todoGather:%v", err)
					gatherErr = fmt.Errorf("todoGather:find errors. open or close or scan error.\n")
				}
				if tmp != nil {
					todoMap[filepath.Join(dirname, info.Name())] = tmp
				}
			}
		}
	}
	return todoMap, gatherErr
}
// show list!
func showTODOList(todoMap map[string][]string) {
	for filename, list := range todoMap {
		fmt.Println(filename)
		for _, s := range list {
			fmt.Println(s)
		}
		fmt.Println()
	}
	fmt.Printf("stack files=%v\n", len(todoMap))
}
func main() {
	// test show
	//crawlshow()

	todoMap, err := mainproc()
	if err != nil {
		log.Fatal(err)
	}
	showTODOList(todoMap)
}
