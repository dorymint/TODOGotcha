# TODOGotcha
---
ガッチャ!


## Example
---

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
Defaults
 - target filetype ".go" or ".txt"
 - target keyword "TODO:"
 - search root "./"

Search from current directory.  
Create "TODO List" from search file.  
Show the "TODO List".  

## Option
---
Show the flags and default parameter
```
todogotcha -h
```

```
-root="<Specify search root directory>"
-filetype="<Target file types list>"
-keyword="<Gather target word>"
```

Use option example
```
todogotcha -root ../ \
          -filetype "go c txt" \
          -keyword "NOTE:"
```

## Licence
---
MIT
