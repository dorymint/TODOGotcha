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
L37:", "Specify gather target keyword"),
L58:GOMAXPROCS いらないかも取り敢えず指定できるように
L220:Review
L221:To simple
L227:出来れば (descriptor limits / 2) で値を決めたい
L265:それでも気になるので、速度を落とさずいい方法があれば修正する
L286:Refactor, To simple!
L330:Refactor
L341:Fix to Duplication
L350:Fix to Duplication
L378:エラーログの出し方考える

-----| RESULT |-----
2 files found have the keyword

ALL FLAGS
root="/home/dory/gowork/src/github.com/dory/go-TODOGotcha/todogotcha"
filetype="go txt"
keywrod="TODO: "
sort="off"
result="on"
date="off"
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

**Default option**
 - root     ./
  - Search root
 - filetype   "go txt"
  - Specify filetype
 - keyword  "TODO: "
  - Specify keyword
 - fileList ""
  - Specify targetr file list
 - dirList  ""
  - Specify search target directory
  
  md table test    
  
  | test | test | test |
  | --- | --- | --- |
  | he | kd | kj |

 - result      on
 - recursively on
 - sort        off
 - date        off
 - proc        0
  - set of GOMAXPROCS


**This example is changed default option**
```
todogotcha -root "../../" \
          -filetype "go c cc cpp txt py" \
          -keyword "NOTE: " \
          -date on
```

## Licence
---
MIT
