# TODOGotcha
---
ガッチャ!  

Search from current directory recursively  
Create TODO list from search files  
Show the TODO list  

**NEXT BRANCH**

## Example
---
![gothca](./gotcha.png "gotcha")  
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
| filetype | Specify target filetypes | "go txt" |
| keyword | Specify keyword | "TODO: " |
| file | Specify target file list | "" |
| dir | Specify directory list, is not recursively | "" |
| separator | Specify separator for directoris and files | ; |
| output | Specify output file | "" |
| force | Ignore override confirm [on:off]? | off |
| recursively | Recursively search [on:off]? | on |
| result | Specify result [on:off]? | on |
| sort | Sort for directory name [on:off]? | off |
| date | Add output DATE in result [on:off]? | off |
| line | Specify number of line | 1 |
| proc | Specify GOMAXPROCS | automatic setting |

**This example is changed default option**
```
todogotcha -root "../../" \
          -keyword "NOTE: " \
          -filetype "go c cc cpp txt py" \
          -dir "./path/to/dir1/;../path/to/dir2/" \
          -file "./path/to/file1;../path/to/file2" \
          -date on \
          -proc 2
```

## Licence
---
MIT
