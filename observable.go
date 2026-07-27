// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

// Observable is a single value that notifies its subscribers whenever it
// changes. It is the "property" primitive of the MVVM layer: a ViewModel holds
// Observables, a View binds widgets to them, and neither side references the
// other directly.
//
// Observable is intentionally NOT safe for concurrent use — mutate it on the UI
// goroutine. For an async producer, hand the value to the app's refresh/queue
// and Set it from the UI tick. Keeping it lock-free matches the single-threaded
// UI model and stays allocation-light.
type Observable[T any] struct {
	value     T
	eq        func(a, b T) bool // change test; nil ⇒ always notify
	subs      map[int]func(T)
	nextID    int
	notifying bool
	dirty     bool
}

// NewObservable seeds a value using == as the change test (T must be
// comparable). Setting an equal value is a no-op, which is what keeps two-way
// bindings loop-free.
func NewObservable[T comparable](v T) *Observable[T] {
	return &Observable[T]{value: v, eq: func(a, b T) bool { return a == b }}
}

// NewObservableEq seeds a value with an explicit equality function — use it for
// slice- or struct-valued observables where == does not apply. A nil eq means
// "never equal": every Set notifies (and two-way loop-suppression is defeated,
// so supply an eq for anything you bind two-way).
func NewObservableEq[T any](v T, eq func(a, b T) bool) *Observable[T] {
	return &Observable[T]{value: v, eq: eq}
}

// Get returns the current value.
func (o *Observable[T]) Get() T { return o.value }

// Set assigns v and, when it differs from the current value (per the change
// test), notifies every subscriber with the new value.
//
// Re-entrant Sets are safe: if a subscriber calls Set again (e.g. to normalise
// or clamp), the value is updated and the notification loop runs another pass
// with the latest value rather than recursing — so a validating subscriber
// converges instead of overflowing the stack. Subscribers are expected to
// converge; a pair that endlessly sets each other to different values is a
// caller bug, not a case Observable defends against.
func (o *Observable[T]) Set(v T) {
	if o.eq != nil && o.eq(o.value, v) {
		return
	}
	o.value = v
	if o.notifying {
		o.dirty = true
		return
	}
	o.notifying = true
	for {
		o.dirty = false
		cur := o.value
		for _, fn := range o.subs {
			fn(cur)
		}
		if !o.dirty {
			break
		}
	}
	o.notifying = false
}

// Subscribe registers fn (which is NOT called immediately) and returns a
// function that removes it. Subscribers run in an unspecified order.
func (o *Observable[T]) Subscribe(fn func(T)) (unsubscribe func()) {
	if o.subs == nil {
		o.subs = map[int]func(T){}
	}
	id := o.nextID
	o.nextID++
	o.subs[id] = fn
	return func() { delete(o.subs, id) }
}

// SubscribeChanged registers a value-less observer, called on every change. It
// lets an Observable satisfy Changeable so heterogeneous sources (a string and
// a bool observable, say) can jointly drive a Command's CanExecute.
func (o *Observable[T]) SubscribeChanged(fn func()) (unsubscribe func()) {
	return o.Subscribe(func(T) { fn() })
}

// Changeable is any source that can notify on change without exposing its
// value's type — implemented by Observable[T] and ObservableList[T]. It is the
// type-erased seam BindCanExecute uses to combine mixed-typed sources.
type Changeable interface {
	SubscribeChanged(fn func()) (unsubscribe func())
}
