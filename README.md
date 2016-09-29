# TODOGotcha
---
ガッチャ!  

Search from current directory recursively  
Create "TODO List" from search files  
Show the "TODO List"  

## Example
---
Output from ```todogotcha -keyword "TODO: "```  
```
/home/dory/gowork/src/github.com/dory/go-todogotcha/todogotcha/todogotcha.go
L22:Reconsider name for sortFlag
L28:今はコメントアウト
L57:To simple
L85:Fix from bad implementation
L166:Review
L167:To simple
L208:それでも気になるので、速度を落とさずいい方法があれば修正する
L235:Refactor
L237:To lighten
L246:Fix to Duplication
L277:エラーをログに出すのを関数単位じゃなくmainまでatを付けて持って帰りたい

/home/dory/gowork/src/github.com/dory/go-todogotcha/todogotcha/todogotcha_test.go
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

-----| RESULT |-----
2 files found have the keyword

ALL FLAGS
root="/home/dory/gowork/src/github.com/dory/go-todogotcha"
filetype="go txt"
keywrod="TODO: "
sort="off"
result="on"
```

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

**Defaults options**
 - filetype "go txt"
 - keyword "TODO:"
 - root "./"
 - sort "off"
 - result "on"

**This example is changed default option**

```
todogotcha -root "../" \
          -filetype "go c cc cpp txt py" \
          -keyword "NOTE: "
```  


| Flags | Defaults | Example |
| :-: | :- | :-: |
| root | ./ | -root ./to/dir/path/ |
| filetype | "go txt" | -filetype cpp |
| keyword | "TODO\:" | -keyword "ToDo\: " |
| file | none | -file ./path/to/file1 |
| recursively | on | -recursively off |
| result | on | -result off |
| date | off | -date on |
| sort | off | -date on |


```
-root "<Specify search root directory>"
-filetype "<Target file types list>"
-keyword "<Gather target word>"

-sort="on" or "off"
-result="on" or "off"
```

## Licence
---
MIT
