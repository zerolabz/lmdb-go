package lmdb

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestBarrierHolds(t *testing.T) {
	b := NewBarrier()
	defer b.Close()

	released := int64(0)

	waiter := func(i int) {
		b.WaitForGate(i)
		//vv("goro %v is released", i)
		atomic.AddInt64(&released, 1)
	}

	for i := 0; i < 3; i++ {
		go waiter(i)
	}
	time.Sleep(time.Second)
	r := atomic.SwapInt64(&released, 0)
	if r != 3 {
		panic("open barrier held back goro")
	}
	//vv("good: barrier started open")

	go func() {
		b.BlockAndWaitUntilCountAtGate(4)
	}()
	for i := 0; i < 3; i++ {
		go waiter(i)
	}

	time.Sleep(time.Second)
	r = atomic.SwapInt64(&released, 0)
	if r != 0 {
		panic("bad: barrier did not hold back goro")
	}
	//vv("good: barrier of 4 did not release on 3")
	waiter(4)
	//vv("good: barrier of 4 released on 4")

}
