package lmdb

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestBarrierHolds(t *testing.T) {
	b := NewBarrier()
	defer b.Close()

	released := int64(0)

	waiter := func(i int) {
		b.WaitAtGate(i)
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

	seenAll := make(chan bool)
	go func() {
		b.BlockUntil(4)
		vv("we have seen 4 waiters")
		close(seenAll)
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
	go waiter(4)
	<-seenAll
	//vv("good: barrier of 4 counted all 4")

	time.Sleep(time.Second)
	r = atomic.SwapInt64(&released, 0)
	if r != 0 {
		panic(fmt.Sprintf("bad: barrier did not hold back goro, should wait for unblock. r = %v", r))
	}

	b.UnblockReaders()

	time.Sleep(time.Second)
	r = atomic.SwapInt64(&released, 0)
	if r != 4 {
		panic("bad: unblock should have released 4 goro")
	}

}
