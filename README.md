# todogotcha
---
ガッチャ!  

Search from current directory recursively  
Create to-do list from search files  
Show the to-do list  

just like find, grep

## Example
---
Output from `todogotcha`  
![gotcha](./gotcha.png "gotcha")  

List from "func "  
`todogotcha -keyword "func " -trim=false`  
![gotcha2](./gotcha2.png "gotcha2")  

## Installation
---
```
go get github.com/dorymint/go-todogotcha/todogotcha
```

## Usage
---
Display the found to-do list like example
```
todogotcha
```

If you need output to file
```
todogotcha -output "./path/to/file"
```

## Option
---
**Show the flags and default parameter**
```
todogotcha -h
```

| Flags | Description | Default |
| :---- | :---------- | :------ |
| root  | Search root | ./ |
| filetype | Target filetypes(suffix) | ".go .txt" |
| keyword | Specify target | "TODO: " |
| file | Specify target files | "" |
| dir | Specify directory list, is do not recursive | "" |
| separator | separator for Flags(dir and file) | ; |
| output | Output filepath | "" |
| force | Ignore override confirm [true:false]? | false |
| recursively | Recursive search from root [true:false]? | true |
| ignore-long | If true, ignore file that has long line [true:false]? | true |
| result | Result for flags state [true:false]? | false |
| sort | For output [true:false]? | false |
| date | Add date [true:false]? | false |
| trim | Trim the keyword from output [true:false]? | true |
| line | Specify number of lines for gather from the keyword | 1 |
| limit | Specify limit of goroutine, for file descriptor | 512 |
| proc | Specify GOMAXPROCS(0 that means automatic) | 0 |

**This example is changed default options**
```
todogotcha -root="./path/to/search/root/" \
          -recursively=false \
          -trim=false \
          -keyword="NOTE: " \
          -line=2 \
          -filetype=".cpp .py .txt .go .vim" \
          -separator=";" \
          -dir="./path/to/dir1/;../path/to/dir2/" \
          -file="./path/to/file1;../path/to/file2" \
          -date=true \
          -proc=1 \
          -result=true
```

## Licence
---
MIT
