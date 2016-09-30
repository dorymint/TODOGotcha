# TODOGotcha
---
ガッチャ!  

Search from current directory recursively  
Create "TODO List" from search files  
Show the "TODO List"  

## Example
---
```
/home/dory/gowork/src/github.com/dory/go-TODOGotcha/todogotcha/todogotcha_test.go
L89:To simple! delete this?
L106:To simple!!
L211:Test
L233:TODO: TODO:
L234:2line
L237:TODO:",
L267:add test case
L312:Create test data and run
L315:Add another case
L327:Add another case
L333:Add another case

/home/dory/gowork/src/github.com/dory/go-TODOGotcha/todogotcha/todogotcha.go
L30:Maby future delete this
L38:", "Specify gather target keyword"),
L47:Maby future delete this
L59:GOMAXPROCS maby future delete this
L148:ここまでロックするならスレッドを分ける意味は薄いかも、再考する
L224:Review, To simple
L230:出来れば (descriptor limits / 2) で値を決めたい
L268:それでも気になるので、速度を落とさずいい方法があれば修正する
L289:Refactor, To simple!
L333:Refactor
L344:Fix to Duplication
L353:Fix to Duplication
L380:Maybe future delete this
L389:エラーログの出し方考える

-----| RESULT |-----
2 files found have the keyword

ALL FLAGS
root="/home/dory/gowork/src/github.com/dory/go-TODOGotcha"
filetype="go txt"
keywrod="TODO: "
sort=off
srecursively=on
result=on
date=off
dirList=[]
fileList=[]
proc=4
```
Output from ```todogotcha```  

## Installation
---
```
go get github.com/dorymint/go-TODOGothca/todogotcha
```

## Usage
---
Display the found TODO list like example
```
todogotcha
```

If you need output to file
```
todogotcha > ./TODOList.log
```

## Option
---
**Show the flags and default parameter**
```
todogotcha -h
```

**Option**

| Flags | Description | Default |
| :---- | :---------- | :------ |
| root  | Search root | -root ./ |
| filetype | Specify target filetypes | -filetype "go txt" |
| keyword | Specify keyword | -keyword "TODO: " |
| file | Specify target file list | -file "" |
| dir | Specify directory list, is not recursively | -dir "" |
| result | Specify result [on:off]? | -result on |
| recursively | Recursively search [on:off]? | -recursively on |
| sort | Sort for directory name [on:off]? | -sort off |
| date | Add output DATE in result [on:off]? | -date off |
| proc | Specify GOMAXPROCS | automatic setting |

**This example is changed default option**
```
todogotcha -root "../../" \
          -keyword "NOTE: " \
          -filetype "go c cc cpp txt py" \
          -dir "path/to/dir1/ ; path/to/dir2/" \
          -file "path/to/file1 ; path/to/file2" \
          -date on \
          -proc 2
```

## Licence
---
MIT
