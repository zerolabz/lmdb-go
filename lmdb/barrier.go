package lmdb

import (
	"github.com/glycerine/idem"
)

// Barrier allows us to temporarily halt all readers, so that
// a writer can commit alone and thus compact the db.
// The Barrier starts unblocked, alllowing passage to any
// caller of WaitForGate().
type Barrier struct {
	wait       chan *appointment // send upon entering the waiting room.
	halt       *idem.Halter
	blockReqCh chan *blockReq
}

type appointment struct {
	id   int
	done chan struct{}
}

func newAppointment(id int) *appointment {
	return &appointment{
		id:   id,
		done: make(chan struct{}),
	}
}

// NewBarrier is either open, allowing immediate passage,
// or blocked, halting all callers at WaitForGate()
// until the barrier is opened.
//
// Barrier.Close() must be called when the barrier
// is no longer needed to avoid a goroutine leak.
func NewBarrier() (b *Barrier) {
	b = &Barrier{
		wait:       make(chan *appointment), // waiters indicate they are waiting for the gate by sending here.
		halt:       idem.NewHalter(),
		blockReqCh: make(chan *blockReq),
	}
	go func() {
		defer b.halt.Done.Close()

		var waitlist []*appointment
		var curBlockReq *blockReq

		for {
			select {
			case br := <-b.blockReqCh:
				if br.count == 0 {
					close(br.done)
					continue
				}
				if curBlockReq == nil {
					// good, changing state from open to closed barrier.
				} else {
					panic("got 2nd block request atop of first")
				}
				curBlockReq = br
				if len(waitlist) != 0 {
					panic("had waiters when we were open, internal/client bug")
				}
			case appt := <-b.wait:
				if curBlockReq == nil {
					close(appt.done)
					continue
				}
				waitlist = append(waitlist, appt)
				if len(waitlist) >= curBlockReq.count {
					for _, appt := range waitlist {
						close(appt.done)
					}
					waitlist = nil
					close(curBlockReq.done)
					curBlockReq = nil
				}
			case <-b.halt.ReqStop.Chan:
				return
			}
		}
	}()
	return
}

func (b *Barrier) WaitForGate(id int) {
	appt := newAppointment(id)
	select {
	case b.wait <- appt:
		<-appt.done
	case <-b.halt.ReqStop.Chan:
	}
}

func (b *Barrier) Close() {
	b.halt.ReqStop.Close()
	<-b.halt.Done.Chan
}

// BlockAndWaitUntilCountAtGate is called with a count, the
// number of waiters required to be present and waiting
// at the gate before call returns.
// A count of <= 0 will return immediately without
// checking the barrier. Otherwise we raise the barrier
// and wait until we have seen count other goroutines waiting
// on it.
func (b *Barrier) BlockAndWaitUntilCountAtGate(count int) {
	if count <= 0 {
		return
	}
	req := newBlockReq(count)
	b.blockReqCh <- req
	<-req.done
}

type blockReq struct {
	count int
	done  chan struct{}
}

func newBlockReq(count int) *blockReq {
	return &blockReq{
		count: count,
		done:  make(chan struct{}),
	}
}
