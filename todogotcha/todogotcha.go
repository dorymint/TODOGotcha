package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Flags for pkg name sort
type Flags struct {
	// Flags for Data
	root        *string
	suffix      *string
	keyword     *string
	fileList    *string
	dirList     *string
	separator   *string
	recursively *string

	// Flags for Output
	result *string
	output *string
	force  *string
	sort   *string
	date   *string

	// TODO: Specify GOMAXPROCS. Maybe future delete this
	proc *int
}

// For stringer, ALL flags
func (f Flags) String() string {
	tmp := fmt.Sprintln("ALL FLAGS")
	tmp += fmt.Sprintf("result=%v\n", *f.result)
	tmp += fmt.Sprintf("root=%q\n", *f.root)
	tmp += fmt.Sprintf("keywrod=%q\n", *f.keyword)
	tmp += fmt.Sprintf("filetype=%q\n", *f.suffix)

	tmp += fmt.Sprintf("recursively=%v\n", *f.recursively)
	tmp += fmt.Sprintf("sort=%v\n", *f.sort)
	tmp += fmt.Sprintf("date=%v\n", *f.date)
	tmp += fmt.Sprintf("force=%v\n", *f.force)

	tmp += fmt.Sprintf("output=%v\n", *f.output)

	tmp += fmt.Sprintf("dirList=%q\n", *f.dirList)
	tmp += fmt.Sprintf("fileList=%q\n", *f.fileList)
	tmp += fmt.Sprintf("separator=%q\n", *f.separator)

	tmp += fmt.Sprintf("proc=%v\n", runtime.GOMAXPROCS(0))
	return tmp
}

var (
	flags = &Flags{
		root:    flag.String("root", "./", "Specify recursively-search root directory"),
		suffix:  flag.String("filetype", "go txt", `Specify target file types into the " "`),
		keyword: flag.String("keyword", "TODO: ", "Specify gather target keyword"),

		fileList:  flag.String("file", "", `Specify file list`),
		dirList:   flag.String("dir", "", `Specify directory list, This want not recursively serach`),
		separator: flag.String("separator", ";", "Specify separator for specify directories and files"),

		output: flag.String("output", "", "Specify output file"),
		force:  flag.String("force", "off", "Ignore override confirm [on:off]?"),

		recursively: flag.String("recursively", "on", `If this "off", not recursively search [on:off]?`),
		result:      flag.String("result", "on", "Specify result [on:off]?"),
		sort:        flag.String("sort", "off", "Specify sorted flags [on:off]?"),
		date:        flag.String("date", "off", "Add output DATE in result [on:off]?"),

		proc: flag.Int("proc", 0, "Specify GOMAXPROCS"),
	}

	suffixList     []string
	fileList       []string
	dirList        []string
	outputFilePath string
)

// TODO: To simple!
// フラグ処理を入れてみたけどinitである必要がない
func init() {
	flag.Parse()

	runtime.GOMAXPROCS(*flags.proc)

	if *flags.root != "" {
		tmp, err := filepath.Abs(*flags.root)
		if err != nil {
			log.Fatalf("init:%v", err)
		} else {
			*flags.root = tmp
		}
	}

	// For output filepath
	if *flags.output != "" {
		cleanpath, err := filepath.Abs(filepath.Clean(strings.TrimSpace(*flags.output)))
		if err != nil {
			log.Fatalf("init:%v", err)
		}
		if _, errstat := os.Stat(cleanpath); errstat == nil && *flags.force == "off" {
			if !ask(fmt.Sprintf("Override? %v", cleanpath)) {
				os.Exit(1)
			}
			// touch
			tmp, err := os.Create(cleanpath)
			if err != nil {
				log.Fatalf("init:%v", err)
			}
			defer func() {
				if errclose := tmp.Close(); errclose != nil {
					log.Fatalf("init:%v", errclose)
				}
			}()
		}
		outputFilePath = cleanpath
	}

	// For specify filetype
	suffixList = strings.Split(*flags.suffix, " ")

	// For specify files and dirs
	pathClean := func(str *[]string, in *string) {
		*str = append(*str, strings.Split(*in, *flags.separator)...)
		for i, s := range *str {
			cleanPath, err := filepath.Abs(filepath.Clean(strings.TrimSpace(s)))
			if err != nil {
				log.Printf("init:%v", err)
				continue
			}
			(*str)[i] = cleanPath
		}
	}
	if *flags.fileList != "" {
		pathClean(&fileList, flags.fileList)
	}
	if *flags.dirList != "" {
		pathClean(&dirList, flags.dirList)
	}

	// Unknown flag check
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

func ask(s string) bool {
	fmt.Println(s)
	fmt.Printf("[yes:no]? >>")
	for sc, i := bufio.NewScanner(os.Stdin), 0; sc.Scan() && i < 2; i++ {
		if sc.Err() != nil {
			log.Fatal(sc.Err())
		}
		switch sc.Text() {
		case "yes":
			return true
		case "no":
			return false
		default:
			fmt.Println(sc.Text())
			fmt.Printf("[yes:no]? >>")
		}
	}
	return false
}

// Use wait group dirsCrawl
// Recursively search
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
		// NOTE: ここまでロックするならスレッドを分ける意味は薄いかも、再考する

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

// For dirsCrawl and specify filepath
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

func suffixSeacher(filename string, targetSuffix []string) bool {
	for _, x := range targetSuffix {
		if strings.HasSuffix(filename, "."+x) {
			return true
		}
	}
	return false
}

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

// NOTE: gopher増やしまくるとcloseが間に合わなくてosのfile descriptor上限に突っかかる
// goroutine にリミットを付けてファイルオープンを制限して上限に引っかからない様にしてみる
// TODO: Review, To simple
func unlimitedGopherWorks(infoMap map[string][]os.FileInfo, filetypes []string, keyword string) (todoMap map[string][]string) {

	todoMap = make(map[string][]string)

	// NOTE: Countermove "too many open files"!!
	// TODO: 出来れば (descriptor limits / 2) で値を決めたい
	// 環境依存のリミットを取得する良い方法を見つけてない(´・ω・`)
	gophersLimit := 512 // NOTE: This Limit is require (Limit < file descriptor limits)
	var gophersLimiter int

	mux := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	// Call gather() and append in todoMap
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

				// NOTE: Countermove "too many open files"
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

// GophersProc generate TODOMap from file list! gatcha!!
// TODO: Refactor, To simple!
func GophersProc(root string) (todoMap map[string][]string) {
	infoMap := make(map[string][]os.FileInfo)

	// For recursively switch
	if *flags.recursively == "on" {
		infoMap = dirsCrawl(root)
	} else {
		infos, err := getInfos(root)
		if err != nil {
			log.Printf("GophersProc:%v", err)
		} else {
			infoMap[root] = infos
		}
	}

	// For specify dirs
	for _, dirname := range dirList {
		if _, ok := infoMap[dirname]; !ok {
			infos, err := getInfos(dirname)
			if err != nil {
				log.Printf("GophersProc:%v", err)
				continue
			}
			infoMap[dirname] = infos
		}
	}

	// Generate todo list from infoMap
	todoMap = unlimitedGopherWorks(infoMap, suffixList, *flags.keyword)

	// For specify files
	for _, s := range fileList {
		if _, ok := todoMap[s]; !ok {
			todoList := gather(s, *flags.keyword)
			if todoList != nil {
				todoMap[s] = todoList
			}
		}
	}
	return todoMap
}

// OutputTODOList is output crawl results
// TODO: Refactor
func OutputTODOList(todoMap map[string][]string) {
	// For Specify output file
	stdout := os.Stdout
	var err error
	if outputFilePath != "" {
		stdout, err = os.Create(outputFilePath)
		if err != nil {
			log.Fatalf("OutputTODOList:%v", err)
		}
		defer func() {
			if errclose := stdout.Close(); errclose != nil {
				log.Printf("OutputTODOList:%v", errclose)
			}
		}()
	}

	// For sort
	if *flags.sort == "on" {
		// Optional
		var filenames []string
		for filename := range todoMap {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)

		// TODO: Fix to Duplication
		for _, filename := range filenames {
			fmt.Fprintln(stdout, filename)
			for _, todo := range todoMap[filename] {
				fmt.Fprintln(stdout, todo)
			}
			fmt.Fprint(stdout, "\n")
		}
	} else {
		// TODO: Fix to Duplication
		for filename, todoList := range todoMap {
			fmt.Fprintln(stdout, filename)
			for _, s := range todoList {
				fmt.Fprintln(stdout, s)
			}
			fmt.Fprint(stdout, "\n")
		}
	}

	if *flags.result == "on" {
		fmt.Fprintln(stdout, "-----| RESULT |-----")
		fmt.Fprintf(stdout, "%v files found have the %q\n\n", len(todoMap), *flags.keyword)

		fmt.Fprintln(stdout, flags)
	}
	if *flags.date == "on" {
		fmt.Fprintf(stdout, "\nDATE:%v\n", time.Now())
	}
}

// TODO: エラーログの出し方を考えたい
// NOTE: flagに直接触れるのは init, main, データトップのGophersProc, 表示トップのOutputTODOList, に限定してみる
func main() {
	todoMap := GophersProc(*flags.root)
	OutputTODOList(todoMap)
}
