# todogotcha
---
ガッチャ!  

Search from current directory recursively  
Create to-do list from search files  
Output the to-do list  

just like find, grep

## Example
---
Output from `todogotcha`  
![gotcha](./gotcha.png "gotcha")  

List from "func "  
`todogotcha -word="func " -trim=false`  
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
todogotcha -out="./path/to/file"
```

If you need search from only pwd
```
todogotcha -recursive=false
```

If you need only specify directory
```
todogotcha -recursive=false -root="/path/to/dir/;/path/to/another/dir/"
```

If you need specify file
```
todogotcha -root="" -file="/path/to/file;/path/to/another/file"
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
| type | Target filetypes(suffix) | ".go .txt" |
| word | Target word | "TODO: " |
| file | Specify target files | "" |
| dir | Specify directory list, is do not recursive | "" |
| sep | Separator for Flags(-dir -file) | ; |
| out | Output to filepath | "" |
| force | Ignore override confirm for Flags(-out) [true:false]? | false |
| recursive | Recursive search from Flags(-root) [true:false]? | true |
| ignore-long | If true, ignore file that has long line [true:false]? | true |
| result | Result for flags state [true:false]? | false |
| sort | For output [true:false]? | false |
| date | Add date [true:false]? | false |
| trim | Trim the keyword from output [true:false]? | true |
| lines | Specify number of lines for gather from the -word | 1 |
| limit | Specify limit of goroutine, for file descriptor | 512 |
| proc | Specify GOMAXPROCS(0 that means automatic) | 0 |

**This example is changed default options**
```
todogotcha -root="./path/to/search/root/" \
          -recursive=false \
          -trim=false \
          -word="NOTE: " \
          -lines=2 \
          -type=".cpp .py .txt .go .vim" \
          -sep=";" \
          -dir="./path/to/dir1/;../path/to/dir2/" \
          -file="./path/to/file1;../path/to/file2" \
          -date=true \
          -proc=1 \
          -result=true
```

## Licence
---
MIT
