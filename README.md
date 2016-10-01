# TODOGotcha
---
ガッチャ!  

Search from current directory recursively  
Create "TODO List" from search files  
Show the "TODO List"  

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
todogotcha > ./TODOList.log
```

## Option
---
**Show the flags and default parameter**
```
todogotcha -h
```

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
          -dir "./path/to/dir1/;../path/to/dir2/" \
          -file "./path/to/file1;../path/to/file2" \
          -date on \
          -proc 2
```

## Licence
---
MIT
