# todogotcha
------------
cli tool for gathering of "TODO: " from files


## Installation
---------------
```
go get github.com/kamisari/go-todogotcha/todogotcha
```

## Example
----------
- `todogotcha` recursive check from current directory
- `todogotcha /path/dir` or `todogotcha -root /path/dir` specify root
- `todogotcha -word "func "` specify target word, default is "TODO: "
- `todogotcha -out /path/log` specify output to file

- `todogotcha -help` print help

## TODO
-------
TODO: update readme

## Licence
----------
MIT
