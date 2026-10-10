// SPDX-License-Identifier: BSD-3-Clause

package mvvm

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Results posted from many goroutines are applied on the draining goroutine,
// each exactly once, with one wake per post; run with -race, this is the
// property.
func TestQueueCarriesWorkToTheDrainingGoroutine(t *testing.T) {
	var wakes atomic.Int64
	q := NewQueue(func() { wakes.Add(1) })
	obs := NewObservable(0)
	const producers, each = 8, 200
	var wg sync.WaitGroup
	for p := range producers {
		wg.Go(func() {
			for i := range each {
				q.Post(func() { obs.Set(p*each + i + 1) })
			}
		})
	}
	ran := 0
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for {
		ran += q.Drain()
		select {
		case <-done:
			ran += q.Drain()
			if ran != producers*each {
				t.Fatalf("%d of %d posted functions ran", ran, producers*each)
			}
			if wakes.Load() != producers*each {
				t.Fatalf("%d wakes for %d posts", wakes.Load(), producers*each)
			}
			if q.Len() != 0 {
				t.Fatalf("%d left", q.Len())
			}
			return
		default:
		}
	}
}

func TestQueueRunsInOrder(t *testing.T) {
	q := NewQueue(nil)
	var got []int
	for i := range 5 {
		q.Post(func() { got = append(got, i) })
	}
	if q.Len() != 5 || q.Drain() != 5 {
		t.Fatal("five posted, five drained")
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("order %v", got)
		}
	}
	if q.Drain() != 0 {
		t.Fatal("drained twice")
	}
}

// A function that posts itself waits for the next Drain: one Drain cannot
// become an endless loop on the UI goroutine.
func TestQueuePostedDuringDrainWaits(t *testing.T) {
	q := NewQueue(nil)
	n := 0
	var again func()
	again = func() { n++; q.Post(again) }
	q.Post(again)
	if q.Drain() != 1 || n != 1 || q.Len() != 1 {
		t.Fatalf("drain ran %d, len %d", n, q.Len())
	}
	q.Drain()
	if n != 2 {
		t.Fatal(n)
	}
}
