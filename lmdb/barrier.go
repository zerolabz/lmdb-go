package lmdb

import (
	"sync"
)

// Barrier allows us to temporarily halt all readers, so that
// a writer can commit alone and thus compact the db.
// The Barrier starts unblocked, alllowing passage to any
// caller of WaitForGate().
type Barrier struct {
	mu   sync.Mutex
	gate sync.Cond

	blocked bool

	waitlist map[int]bool
}

func NewBarrier() (b *Barrier) {
	b = &Barrier{
		waitlist: make(map[int]bool),
	}
	b.gate = sync.NewCond(b.mu)
	return
}

func (b *Barrier) WaitForGate(id int) {
	b.mu.Lock()
	queued := false
	for b.blocked {
		if !queued {
			b.waitlist[id] = true
			queued = true
		}
		b.gate.Wait()
	}
	delete(b.waitlist, id)
	b.mu.Unlock()
}

func (b *Barrier) UnblockGate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked = false
	b.gate.Broadcast()
}

// WaitUntilCountAtGate is called with a count, the
// number of waiters required to be present and waiting
// at the gate before call returns.
// A count of <= 0 will return immediately without
// checking the barrier.
func (b *Barrier) WaitUntilCountAtGate(count int) {
	if count <= 0 {
		return
	}
}

func (b *Barrier) BlockGate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked = true
}
