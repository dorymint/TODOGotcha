package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// flags
var (
	root       = flag.String("root", "./", "Specify search root")
	suffix     = flag.String("filetype", "go txt", `Specify target file types into the " "`)
	suffixList []string
	keyword    = flag.String("keyword", "TODO:", "Specify gather target keyword")
	// TODO: Reconsider name for sortFlag
	sortFlag = flag.String("sort", "off", "Specify output list sorted [on:off]?")
	result   = flag.String("result", "on", "Specify result [on:off]?")
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
func dirsCrawl(root string) map[string][]os.FileInfo {
	// mux group
	DirsCache := make(map[string]bool)
	InfoCache := make(map[string][]os.FileInfo)
	Mux := new(sync.Mutex)

	Wg := new(sync.WaitGroup)

	var crawl func(string)
	crawl = func(dirname string) {
		defer Wg.Done()

		Mux.Lock()
		if DirsCache[dirname] {
			Mux.Unlock()
			return
		}
		DirsCache[dirname] = true

		f, err := os.Open(dirname)
		if err != nil {
			log.Printf("crawl:%v", err)
			Mux.Unlock()
			return
		}
		defer f.Close()

		info, err := f.Readdir(0)
		if err != nil {
			log.Printf("crawl info:%v", err)
			Mux.Unlock()
			return
		}
		InfoCache[dirname] = info

		// NOTE:countermove for too many open files
		if err := f.Close(); err != nil {
			log.Println(err)
		}
		Mux.Unlock()
		// "too many open files" の対応でlockしたけど...
		// ここまでlockするならスレッド分ける意義が薄そう...

		for _, x := range info {
			if x.IsDir() {
				Wg.Add(1)
				go crawl(filepath.Join(dirname, x.Name()))
			}
		}
	}

	Wg.Add(1)
	crawl(root)
	Wg.Wait()
	return InfoCache
}

// Use flag suffixList
func suffixSeacher(filename string, targetSuffix []string) bool {
	for _, x := range targetSuffix {
		if strings.HasSuffix(filename, "."+x) {
			return true
		}
	}
	return false
}

// specify filename and target, Gather target(TODOs), return todoList.
// シンプルでいい感じに見えるけど、goroutineで呼びまくると...(´・ω・`)っ"too many open files"
// TODO: Coutnermove "too many opne files"
// TODO: !!todoListをchannelに変えてstringを投げるようにすれば数を制限したgoroutineが使えそう
func gather(filename string, target string) (todoList []string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("todoGather:%v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("gather:%v", err)
		}
	}()

	for sc, i := bufio.NewScanner(f), uint(1); sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("todoGather:%v", err)
		}
		if index := strings.Index(sc.Text(), target); index != -1 {
			todoList = append(todoList, fmt.Sprintf("L%v:%s", i, sc.Text()[index+len(target):]))
		}
	}
	return todoList, nil
}

// にゃん
// Use flag keyword
func unlimitedGophersWroks(infoMap map[string][]os.FileInfo) (todoMap map[string][]string) {
	// TODO: Use goroutine
	todoMap = make(map[string][]string)

	//mux := new(sync.Mutex)
	//wg := new(sync.WaitGroup)

	worker := func(filepath string) {
		//defer wg.Done()
		// TODO: Countermove "too many open files"
		todoList, err := gather(filepath, *keyword)
		if err != nil {
			log.Println(err)
		}
		if todoList != nil {
			//mux.Lock()
			todoMap[filepath] = todoList
			//mux.Unlock()
		}
	}

	for dirname, infos := range infoMap {
		for _, info := range infos {
			if suffixSeacher(info.Name(), suffixList) {
				// TODO: file open and close to tmp string
				//wg.Add(1)
				//go worker(filepath.Join(dirname, info.Name()))
				worker(filepath.Join(dirname, info.Name()))
			}
		}
	}
	//wg.Wait()
	return todoMap
}
func gophersProc() (todoMap map[string][]string) {
	infomap := dirsCrawl(*root)
	todoMap = unlimitedGophersWroks(infomap)
	return todoMap
}

// output to os.Stdout!
func outputTODOList(todoMap map[string][]string) {
	if *sortFlag == "on" {
		// Optional
		var filenames []string
		for filename := range todoMap {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)

		for _, filename := range filenames {
			fmt.Println(filename)
			for _, todo := range todoMap[filename] {
				fmt.Println(todo)
			}
			fmt.Println()
		}
	} else {
		// Main
		for filename, todoList := range todoMap {
			fmt.Println(filename)
			for _, s := range todoList {
				fmt.Println(s)
			}
			fmt.Println()
		}
	}
	if *result == "on" {
		fmt.Println("-----| RESULT |-----")
		fmt.Printf("find %v files\n\n", len(todoMap))
		fmt.Println("ALL FLAGS")
		fmt.Printf("root=%q\n", *root)
		fmt.Printf("filetype=%q\n", *suffix)
		fmt.Printf("keywrod=%q\n", *keyword)
		fmt.Printf("sort=%q\n", *sortFlag)
		fmt.Printf("result=%q\n", *result)
	}
}
func main() {
	todoMap := gophersProc()
	outputTODOList(todoMap)
}
