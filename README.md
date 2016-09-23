# TODOGotcha
---
ガッチャ!  

Search from current directory recursively.  
Create "TODO List" from search files.  
Show the "TODO List".  

## Example
---
Output from ```todogotcha```  
```
/home/dory/gowork/src/github.com/dory/go-todogotcha/todogotcha/todogotcha.go
TODO:
L14:LIST
L25:", "Specify gather target keyword")
L151:Use goroutine
L160:Caution!!
L185:file open and close to tmp string
L222:Erase after test
L233:erase after imprementertion to goroutine procs!!

/home/dory/gowork/src/github.com/dory/go-todogotcha/todogotcha/todogotcha_test.go
TODO:
L48:for the moment
L198:Create test data tuple list
L201:Test`)
L208:"

-----| RESULT |-----
find 2 files

ALL FLAGS
root="/home/dory/gowork/src/github.com/dory/go-todogotcha"
filetype="go txt"
keywrod="TODO:"
result="on"
```

## Installation
---
```
go get github.com/dorymint/go-TODOGothca/todogotcha
```

## Usage
---
```
todogotcha
```

## Option
---
**Show the flags and default parameter**
```
todogotcha -h
```

**Defaults**
 - target filetype ".go" or ".txt"
 - target keyword "TODO:"
 - search root "./"

**This example is changed default option**
```
todogotcha -root "../" \
          -filetype "go c txt" \
          -keyword "NOTE:"
```

```
-root="<Specify search root directory>"
-filetype="<Target file types list>"
-keyword="<Gather target word>"
```

## Licence
---
MIT
