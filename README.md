# gotcha
------------
cli tool for gathering of "TODO: " from files


## Installation
---------------
```
go get github.com/kamisari/go-todogotcha/gotcha
```

## Example
----------
- `gotcha` recursive check from current directory
- `gotcha /path/dir` or `gotcha -root /path/dir` specify root
- `gotcha -word "func "` specify target word, default is "TODO: "
- `gotcha -out /path/log` specify output to file

- `gotcha -help` print help

## TODO
-------
TODO: update readme

## Licence
----------
MIT
