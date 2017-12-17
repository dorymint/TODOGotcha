package main

import (
	"fmt"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"testing"
)

func BenchmarkAtomic(b *testing.B) {
	var (
		rwmux = new(sync.RWMutex)
		write = func(str string) (int, error) {
			rwmux.Lock()
			defer rwmux.Unlock()
			return fmt.Fprintln(ioutil.Discard, str)
		}
	)

	var (
		u1 uint64
		u2 uint64
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.AddUint64(&u1, 1)
		atomic.AddUint64(&u2, u1)
		write("hi")
	}
}

func BenchmarkMux(b *testing.B) {
	var (
		mux = new(sync.Mutex)
		u1  uint64
		u2  uint64
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.Lock()
		u1++
		u2 += u1
		fmt.Fprintln(ioutil.Discard, "hi")
		mux.Unlock()
	}
}
