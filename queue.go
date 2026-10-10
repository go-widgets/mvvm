// SPDX-License-Identifier: BSD-3-Clause

package mvvm

import "sync"

// A Queue carries work from any goroutine to the UI goroutine.
//
// Observables are not safe for concurrent use, so a result that arrives on
// another goroutine -- a fetch, a timer, a file watcher -- must not Set one
// there. It Posts a function instead; the UI goroutine runs every posted
// function, in order, when it calls Drain, which a host does at the start of a
// frame, before layout.
//
// Posting also calls wake, so that an idle window -- its run loop blocked on
// the next input event -- draws a frame and so drains. A window back-end
// usually offers that as a Repaint method safe from any goroutine
// (go-widgets/window's Repainter). Wakes are not coalesced here: a back-end
// that repaints coalesces them already, and one call per Post is what lets a
// host that does not, still see every one.
//
//	q := mvvm.NewQueue(func() { repainter.Repaint() })
//	go func() {
//		shares, err := api.ListShares(ctx)
//		q.Post(func() { vm.setShares(shares, err) }) // runs on the UI goroutine
//	}()
type Queue struct {
	mu      sync.Mutex
	pending []func()
	wake    func()
}

// NewQueue returns a queue that calls wake after each Post. wake may be nil:
// the posted work then waits for the next frame something else causes.
func NewQueue(wake func()) *Queue { return &Queue{wake: wake} }

// Post queues fn to run on the UI goroutine, and wakes the host. Safe to call
// from any goroutine; it never runs fn itself.
func (q *Queue) Post(fn func()) {
	q.mu.Lock()
	q.pending = append(q.pending, fn)
	q.mu.Unlock()
	if q.wake != nil {
		q.wake()
	}
}

// Drain runs, on the calling goroutine and in the order they were posted, the
// functions queued so far, and reports how many ran. Call it from the UI
// goroutine only. A function posted while Drain runs -- by one of the drained
// functions, or by another goroutine -- waits for the next Drain, so a
// function that posts itself cannot hold the UI goroutine forever.
func (q *Queue) Drain() int {
	q.mu.Lock()
	run := q.pending
	q.pending = nil
	q.mu.Unlock()
	for _, fn := range run {
		fn()
	}
	return len(run)
}

// Len is how many functions are waiting for the next Drain.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}
