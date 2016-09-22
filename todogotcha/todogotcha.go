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
)

// TODO:LIST
// parse flags
// search dirs
// search in files
// show result

// flags
var (
	root         = flag.String("root", "./", "Specify search root")
	suffix       = flag.String("filetype", "go txt", `Specify target file type into the " "`)
	suffixList   []string
	gatherTarget = flag.String("key", "TODO:", "Specify gather target keyword")
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
func dirsCrawl(root string) (map[string][]os.FileInfo) {
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
			return
		}
		DirsCache[dirname] = true
		Mux.Unlock()

		f, err := os.Open(dirname)
		if err != nil {
			log.Printf("crawl:%v\n", err)
			return
		}
		defer f.Close()

		info, err := f.Readdir(0)
		if err != nil {
			log.Printf("crawl info:%v", err)
			return
		}
		Mux.Lock()
		InfoCache[dirname] = info
		Mux.Unlock()

		// NOTE:countermove for too many open files
		if err := f.Close(); err != nil {
			log.Println(err)
		}

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
	// DirsList は range InfoCache で取得できるので省いた方がいいかも
	return InfoCache
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
func gather(filename string, target string) ([]string, error) {
	todoList := []string{target}
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
			todoList = append(todoList, fmt.Sprintf("L%v:%s", i, sc.Text()[index+len(target):]))
		}
	}

	if len(todoList) == 1 {
		return nil, nil
	}
	return todoList, nil
}

// にゃん
func unlimitedGophersWroks(infoMap map[string][]os.FileInfo) (todoMap map[string][]string) {
	// need use goroutine
	todoMap = make(map[string][]string)
	mux := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	worker := func(filepath string) {
		defer wg.Done()
		// TODO:!!
		// ファイルが多いとosのファイルディスクリプタ上限で叩かれるぅ...どうしよ...
		// スレッドを増やしまくるgoだとよくハマるっぽい
		// wgでカウント取って上限でwaitできれば良さそうだけど、カウンタは公開されてない
		// muxと自前のカウンタで何とかするか...ちょっと考える
		// 取り敢えず簡単にカウント取って一定値でwaitする?
		// os環境個別のディスクリプタ上限を取得する関数とかあれば管理できそうだけど見つかってぬぃ...
		// これをクリアできないと大量のファイルを同時に走査するマルチスレッド使えない...
		// 新しいルーチン呼ぶ前にファイルの内容をバッファに投げてクローズすれば良さそう?
		// バッファがメモリ圧迫しそう...
		// strings.Newreader(tmp)を渡そう...
		todoList, err := gather(filepath, *gatherTarget)
		if err != nil {
			log.Println(err)
		}
		if todoList != nil {
			mux.Lock()
			todoMap[filepath] = todoList
			mux.Unlock()
		}
	}

	for dirname, infos := range infoMap {
		for _, info := range infos {
			if suffixSeacher(info.Name(), suffixList) {
				// TODO:file open and close to tmp string
				wg.Add(1)
				go worker(filepath.Join(dirname, info.Name()))
			}
		}
	}
	wg.Wait()
	return todoMap
}
func useGophersProc() (todoMap map[string][]string) {
	infomap := dirsCrawl(*root)
	todoMap = unlimitedGophersWroks(infomap)
	return todoMap
}

// TODOGotcha!! main proc
// 驚くべき読みにくさ...何とかしたい
// 多分データ構造の選択を間違えてる
// TODO:erase after imprementertion to goroutine procs!!
// do not used
func mainproc() (todoMap map[string][]string, gatherErr error) {
	todoMap = make(map[string][]string)
	infomap := dirsCrawl(*root)
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

// output list!
func outputTODOList(todoMap map[string][]string) {
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

	//	todoMap, err := mainproc()
	//	if err != nil {
	//		log.Fatal(err)
	//	}

	todoMap := useGophersProc()
	outputTODOList(todoMap)
}
