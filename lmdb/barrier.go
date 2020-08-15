package lmdb

import (
	"sync"
)

// Barrier allows us to temporarily halt all readers, so that
// a writer can commit alone and thus compact the db.
// The Barrier starts unblocked, alllowing passage to any
// caller of WaitForGate(). Writers can BlockGate()
// and then wait for a minimum number of waiters by
// calling UnblockGate with a waitCount of the number
// of waiters to require being present before they are
// all released. A waitCount of 0 will release the
// gate immediately and return. A waitCount > 0 means
// that UnblockGate will wait until a count of goroutines
// at WaitForGate has reached >= the waitCount.
type Barrier struct {
	mu   sync.Mutex
	cond sync.Cond

	blocked bool

	waitlist []int
}

func NewBarrier() (b *Barrier) {
	b = &Barrier{
		fleetsz: fleetsz,
	}
	b.cond = sync.NewCond(b.mu)
	return
}

func (b *Barrier) WaitForGate() {
	b.mu.Lock()
	for b.blocked {
		b.cond.Wait()
	}
	b.mu.Unlock()
}

func (b *Barrier) UnblockGate(waitCount int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked = false
	b.cond.Broadcast()
}

func (b *Barrier) BlockGate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked = true
}
