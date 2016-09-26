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

// flags
var (
	root       = flag.String("root", "./", "Specify search root directory")
	suffix     = flag.String("filetype", "go txt", `Specify target file types into the " "`)
	suffixList []string
	keyword    = flag.String("keyword", "TODO:", "Specify gather target keyword")
	// TODO: Reconsider name for sortFlag
	sortFlag = flag.String("sort", "off", "Specify sorted flags [on:off]?")
	result   = flag.String("result", "on", "Specify result [on:off]?")
)

func init() {
	// TODO: 今はコメントアウト
	// runtime.GOMAXPROCS(runtime.NumCPU())

	var err error
	flag.Parse()
	*root, err = filepath.Abs(*root)
	if err != nil {
		log.Fatalf("init:%v", err)
	}
	suffixList = strings.Split(*suffix, " ")
	argsCheck()
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

// Use wait group!!
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

		mux.Lock()
		if dirsCache[dirname] {
			mux.Unlock()
			return
		}
		dirsCache[dirname] = true

		f, err := os.Open(dirname)
		if err != nil {
			log.Printf("crawl:%v", err)
			mux.Unlock()
			return
		}
		// Case of f.Readdir error use this
		defer func() {
			// TODO: Fix from bad implementation
			// os.Invalid == (f == nil)
			// This comparison is maybe bad implementation...
			errclose := f.Close()
			if errclose != nil && errclose.Error() != os.ErrInvalid.Error() {
				log.Printf("crawl:%v", errclose)
			}
		}()

		info, err := f.Readdir(0)
		if err != nil {
			log.Printf("crawl info:%v", err)
			mux.Unlock()
			return
		}
		infoCache[dirname] = info

		// NOTE: Countermove for "too many open files"
		if err := f.Close(); err != nil {
			log.Printf("crawl:%v", err)
		}
		mux.Unlock()
		// "too many open files" の対応でlockしたけど...
		// ここまでlockするならスレッド分ける意義が薄そう...

		for _, x := range info {
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
// GophersProc is get todoMap data
func GophersProc(root string) (todoMap map[string][]string) {
	infomap := dirsCrawl(root)
	todoMap = unlimitedGopherWorks(infomap, suffixList, *keyword)
	return
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
		// Main ...duplication
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

// TODO: エラーをログに出すのを関数単位じゃなくmainまでatを付けて持って帰りたい
// NOTE: fmt.Errorf()でatを入れて返すとエラーのタイプが変わる
func main() {
	todoMap := GophersProc(*root)
	OutputTODOList(todoMap)
}
