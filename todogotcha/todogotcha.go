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
	"time"
)

// TODO: Add flag specify search files
// TODO: Add flag specify search directory, is do not recursively

// flags
var (
	root            = flag.String("root", "./", "Specify search root directory")
	suffix          = flag.String("filetype", "go txt", `Specify target file types into the " "`)
	suffixList      []string
	keyword         = flag.String("keyword", "TODO:", "Specify gather target keyword")
	fileListFlag    = flag.String("file", "", `Specify files path separator is ";" "/path/to/file1;/path/to/file2"`)
	specifyFileList []string
	dirsFlag        = flag.String("dir", "", "Specify directory, This want not recursively serach")
	specifyDirList  []string
	// TODO: Reconsider name for sortFlag
	recursively = flag.String("recursively", "on", `If this "off", not recursively search`)
	sortFlag    = flag.String("sort", "off", "Specify sorted flags [on:off]?")
	result      = flag.String("result", "on", "Specify result [on:off]?")
	date        = flag.String("date", "off", "Add output DATE in result [on:off]?")
)

func init() {
	// TODO: GOMAXPROCS 今はコメントアウト
	//runtime.GOMAXPROCS(runtime.NumCPU())

	var err error
	flag.Parse()
	*root, err = filepath.Abs(*root)
	if err != nil {
		log.Fatalf("init:%v", err)
	}
	suffixList = strings.Split(*suffix, " ")
	argsCheck()

	if *fileListFlag != "" {
		specifyFileList = append(specifyFileList, strings.Split(*fileListFlag, ",")...)
	}
	if *dirsFlag != "" {
		specifyDirList = append(specifyDirList, strings.Split(*dirsFlag, ",")...)
	}
}

// Checking after parsing flags
func argsCheck() {
	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "cmd = %v\n\n", os.Args)
		fmt.Fprintf(os.Stderr, "-----| Unknown option |-----\n\n")
		for _, x := range flag.Args() {
			fmt.Fprintln(os.Stderr, x)
		}
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintln(os.Stderr, "-----| Usage |-----")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

// Use wait group dirsCrawl
// TODO: To simple
func dirsCrawl(root string) map[string][]os.FileInfo {
	// mux group
	dirsCache := make(map[string]bool)
	infoCache := make(map[string][]os.FileInfo)
	mux := new(sync.Mutex)

	wg := new(sync.WaitGroup)

	var crawl func(string)
	crawl = func(dirname string) {
		defer wg.Done()
		infos := new([]os.FileInfo)

		// NOTE: Countermove "too many open files"
		mux.Lock()
		ok := func() bool {
			if dirsCache[dirname] {
				return false
			}
			dirsCache[dirname] = true

			var err error
			*infos, err = getInfos(dirname)
			if err != nil {
				log.Printf("crawl:%v", err)
				return false
			}
			infoCache[dirname] = *infos
			return true
		}()
		mux.Unlock()
		if !ok {
			return
		}
		// NOTE: ここまでロックするならスレッドを分ける意味は薄いかも

		for _, x := range *infos {
			if x.IsDir() {
				wg.Add(1)
				go crawl(filepath.Join(dirname, x.Name()))
			}
		}
	}

	wg.Add(1)
	crawl(root)
	wg.Wait()
	return infoCache
}

// for dirsCrawl and specify filepath
func getInfos(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, fmt.Errorf("getInfos:%v", err)
	}
	defer func() {
		if errclose := f.Close(); errclose != nil {
			log.Printf("getInfos:%v", errclose)
		}
	}()
	infos, err := f.Readdir(0)
	if err != nil {
		return nil, fmt.Errorf("getInfos:%v", err)
	}
	return infos, nil
}

// Use suffixList []string
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
// REMIND: todoListをchannelに変えてstringを投げるようにすれば数を制限したgoroutineが使えそう
func gather(filename string, target string) (todoList []string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("gather:%v", err)
		return nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("gather:%v", err)
		}
	}()

	sc := bufio.NewScanner(f)
	for i := uint(1); sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			log.Printf("gather:%v", err)
			return nil
		}
		if index := strings.Index(sc.Text(), target); index != -1 {
			todoList = append(todoList, fmt.Sprintf("L%v:%s", i, sc.Text()[index+len(target):]))
		}
	}
	return todoList
}

// Use flag keyword
// NOTE:
// gopher増やしまくるとcloseが間に合わなくてosのfile descriptor上限に突っかかる
// goroutine にリミットを付けてファイルオープンを制限して上限に引っかからない様にしてみる
// TODO: Review
// TODO: To simple
func unlimitedGopherWorks(infoMap map[string][]os.FileInfo, filetypes []string, keyword string) (todoMap map[string][]string) {

	todoMap = make(map[string][]string)

	// NOTE: Countermove "too many open files"!!
	// TODO: 出来れば (descriptor limits / 2) で値を決めたい
	// 環境依存のリミットを取得する方法を探す
	gophersLimit := 512 // NOTE: This Limit is requir (Limit < file descriptor limits)
	var gophersLimiter int

	mux := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	// call gather() and append in todoMap
	worker := func(filepath string) {
		defer wg.Done()
		defer func() {
			mux.Lock()
			gophersLimiter--
			mux.Unlock()
		}()

		todoList := gather(filepath, keyword)
		if todoList != nil {
			mux.Lock()
			todoMap[filepath] = todoList
			mux.Unlock()
		}
	}

	for dirname, infos := range infoMap {
		for _, info := range infos {
			if suffixSeacher(info.Name(), filetypes) {
				wg.Add(1)
				mux.Lock()
				gophersLimiter++
				mux.Unlock()

				go worker(filepath.Join(dirname, info.Name()))

				// NOTE:
				// Countermove "too many open files"
				// gophersLimiterの読み出しで値が不確定だけどこれは大体合ってれば問題ないはず
				// TODO: それでも気になるので、速度を落とさずいい方法があれば修正する
				if gophersLimiter > gophersLimit/2 {
					time.Sleep(time.Microsecond)
				}
				if gophersLimiter > gophersLimit {
					log.Printf("Open files %v over, Do limitation to Gophers!!", gophersLimit)
					log.Printf("Wait gophers...")
					wg.Wait()
					log.Printf("Done!")
					mux.Lock()
					gophersLimiter = 0
					mux.Unlock()
				}
			}
		}
	}
	wg.Wait()
	return todoMap
}

// GophersProc generate TODOMap!
// TODO: switch from flag to specify directory or files
// TODO: split filepathes
// Generate file list
// gatcha
// TODO: (´・ω・`)
func GophersProc(root string) (todoMap map[string][]string) {
	infomap := new(map[string][]os.FileInfo)

	*infomap = dirsCrawl(root)

	//	if *dirsFlag != "" {
	//		for _, dirname := range specifyDirList {
	//			if _, ok := infomap[dirname]; !ok {
	//				// TODO:
	//			}
	//		}
	//	}

	todoMap = unlimitedGopherWorks(*infomap, suffixList, *keyword)

	// TODO: file specify, Refactor to simple!
	if *fileListFlag != "" {
		for _, s := range specifyFileList {
			filename, err := filepath.Abs(filepath.Clean(strings.TrimSpace(s)))
			if err != nil {
				log.Printf("GophersProc:%v", err)
				break
			}
			if _, ok := todoMap[filename]; !ok {
				todoList := gather(filename, *keyword)
				if todoList != nil {
					todoMap[filename] = todoList
				}
			}
		}
	}
	return todoMap
}

// OutputTODOList is output crawl results
// TODO: Refactor
func OutputTODOList(todoMap map[string][]string) {
	// TODO: To lighten
	if *sortFlag == "on" {
		// Optional
		var filenames []string
		for filename := range todoMap {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)

		// TODO: Fix to Duplication
		for _, filename := range filenames {
			fmt.Println(filename)
			for _, todo := range todoMap[filename] {
				fmt.Println(todo)
			}
			fmt.Println()
		}
	} else {
		// TODO: Fix to Duplication
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
		fmt.Printf("%v files found have the keyword\n\n", len(todoMap))

		fmt.Println("ALL FLAGS")
		fmt.Printf("root=%q\n", *root)
		fmt.Printf("filetype=%q\n", *suffix)
		fmt.Printf("keywrod=%q\n", *keyword)
		fmt.Printf("sort=%q\n", *sortFlag)
		fmt.Printf("result=%q\n", *result)
		fmt.Printf("date=%q\n", *date)
		if *date == "on" {
			fmt.Print("\n")
			fmt.Printf("DATE:%v\n", time.Now())
		}
	}
}

// TODO: エラーログの出し方考える
// NOTE: logを受けるグローバルなチャンネル作ってロガーをinitでgo logger(){ for{log.Print(<-ch)} }してれば軽いかも?
// NOTE: fmt.Errorf()でatを入れて返すとエラーのタイプが変わる
func main() {
	todoMap := GophersProc(*root)
	OutputTODOList(todoMap)
}
