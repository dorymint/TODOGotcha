package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"unicode/utf8"
)

var ErrTooManyLines = errors.New("too many lines")
var ErrUnavailableText = errors.New("unavailable encoding")

type ExpectedError struct {
	path string
	err  error
}

func (e *ExpectedError) Error() string {
	return fmt.Sprintf("ExpectedError:%s:%v", e.path, e.err)
}

type File struct {
	Path     string
	Contexts []*Context
}

type Context struct {
	index int
	lines []*Line
	loc   []int
}

func (c *Context) String() string {
	var s string
	for i, l := range c.lines {
		if i == c.index {
			s += fmt.Sprintf("%d:%s\n", l.Num, l.Str)
			continue
		}
		s += fmt.Sprintf("%d-%s\n", l.Num, l.Str)
	}
	return s
}

type Line struct {
	Num uint
	Str string
}

// remove?
type linesBuffer struct {
	capa int
	buf  []*Line
}

func newLinesBuffer(capa int) *linesBuffer {
	return &linesBuffer{
		capa: capa,
		buf:  make([]*Line, 0, capa),
	}
}
func (lb *linesBuffer) len() int { return len(lb.buf) }
func (lb *linesBuffer) reset()   { lb.buf = lb.buf[:0] }

// believe not out of bound
func (lb *linesBuffer) del() { lb.buf = lb.buf[1:] }
func (lb *linesBuffer) push(l *Line) {
	if lb.capa == len(lb.buf) {
		lb.buf = append(lb.buf[1:], l)
		return
	}
	lb.buf = append(lb.buf, l)
}

func (lb *linesBuffer) popAll() []*Line {
	ls := make([]*Line, lb.len())
	copy(ls, lb.buf)
	lb.buf = lb.buf[:0]
	return ls
}

// TODO: fix
type FileReader struct {
	// change to []*Line?
	lb *linesBuffer

	c  *Context
	cs []*Context

	// number of lines
	nbefore int
	nafter  int

	i    uint   // current number of lines
	loc  []int  // location of matched
	text string // scanned result
	re   *regexp.Regexp

	// for apppend *FileReader.c to *FileReader.cs
	appendFunc func()
}

func NewFileReader(re *regexp.Regexp, nbefore int, nafter int) *FileReader {
	if nbefore < 0 {
		nbefore = 0
	}
	if nafter < 0 {
		nafter = 0
	}
	max := (math.MaxInt16 / 2) - 1 // 16383
	if nbefore > max || nafter > max {
		panic("NewFileReader: out of bound")
	}
	fr := &FileReader{
		lb:      newLinesBuffer(nbefore + 1 + nafter),
		c:       &Context{},
		nbefore: nbefore,
		nafter:  nafter,
		re:      re,
	}
	switch {
	case nbefore == 0 && nafter == 0:
		fr.appendFunc = fr.appendLine
	case nbefore != 0 && nafter != 0:
		fr.appendFunc = fr.appendContext
	case nbefore != 0:
		fr.appendFunc = fr.appendBeforeLines
	case nafter != 0:
		fr.appendFunc = fr.appendAfterLines
	}
	return fr
}

func (fr *FileReader) Reset() {
	fr.lb.reset()
	fr.c = &Context{}
	fr.cs = fr.cs[:0]
	fr.loc = fr.loc[:0]
}

// TODO: fix
func (fr *FileReader) appendLine() {
	if len(fr.loc) == 2 {
		fr.cs = append(fr.cs, &Context{
			index: 0,
			loc:   fr.loc,
			lines: []*Line{{fr.i, fr.text}},
		})
	}
}
func (fr *FileReader) appendContext() {
	if len(fr.c.loc) == 2 {
		if len(fr.loc) == 2 {
			fr.c.lines = append(fr.c.lines, fr.lb.popAll()...)
			fr.cs = append(fr.cs, fr.c)
			fr.c = &Context{
				index: 0,
				lines: []*Line{{fr.i, fr.text}},
				loc:   fr.loc,
			}
			return
		}
		if fr.lb.len() == fr.nafter {
			fr.c.lines = append(fr.c.lines, fr.lb.popAll()...)
			fr.cs = append(fr.cs, fr.c)
			fr.c = &Context{}
			return
		}
		fr.lb.push(&Line{fr.i, fr.text})
		return
	}
	if len(fr.loc) == 2 {
		fr.c.lines = append(fr.lb.popAll(), &Line{fr.i, fr.text})
		fr.c.index = len(fr.c.lines) - 1
		fr.c.loc = fr.loc
		return
	}
	if fr.lb.len() == fr.nbefore {
		fr.lb.del()
	}
	fr.lb.push(&Line{fr.i, fr.text})
}
func (fr *FileReader) appendBeforeLines() {
	if len(fr.loc) == 2 {
		fr.c.lines = append(fr.lb.popAll(), &Line{fr.i, fr.text})
		fr.c.index = len(fr.c.lines) - 1
		fr.c.loc = fr.loc
		fr.cs = append(fr.cs, fr.c)
		fr.c = &Context{}
		return
	}
	if fr.lb.len() == fr.nbefore {
		fr.lb.del()
	}
	fr.lb.push(&Line{fr.i, fr.text})
}
func (fr *FileReader) appendAfterLines() {
	if len(fr.loc) == 2 {
		if len(fr.c.loc) == 2 {
			fr.c.lines = append(fr.c.lines, fr.lb.popAll()...)
			fr.cs = append(fr.cs, fr.c)
			fr.c = &Context{}
		}
		fr.c.index = 0
		fr.c.lines = []*Line{{fr.i, fr.text}}
		fr.c.loc = fr.loc
		return
	} else if len(fr.c.loc) == 2 {
		if fr.lb.len() == fr.nafter {
			fr.c.lines = append(fr.c.lines, fr.lb.popAll()...)
			fr.cs = append(fr.cs, fr.c)
			fr.c = &Context{}
			return
		}
		if fr.lb.len() == fr.nafter {
			fr.lb.del()
		}
		fr.lb.push(&Line{fr.i, fr.text})
	}
}

func (fr *FileReader) ReadFile(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	defer fr.Reset()

	sc := bufio.NewScanner(f)
	for fr.i = uint(1); sc.Scan(); fr.i++ {
		if fr.i == 0 {
			return nil, &ExpectedError{path: path, err: ErrTooManyLines}
		}
		fr.text = sc.Text()
		if !utf8.ValidString(fr.text) {
			return nil, &ExpectedError{path: path, err: ErrUnavailableText}
		}
		fr.loc = fr.re.FindStringIndex(fr.text)
		fr.appendFunc()
	}
	if err = sc.Err(); err != nil {
		if err == bufio.ErrTooLong {
			return nil, &ExpectedError{path: path, err: err}
		}
		return nil, err
	}

	// append last one
	if len(fr.c.loc) == 2 {
		fr.c.lines = append(fr.c.lines, fr.lb.popAll()...)
		fr.cs = append(fr.cs, fr.c)
	}

	file := &File{
		Path:     path,
		Contexts: make([]*Context, len(fr.cs)),
	}
	copy(file.Contexts, fr.cs)
	return file, nil
}
