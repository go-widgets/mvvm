// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

// ListChangeKind classifies an ObservableList change so a bound view can update
// incrementally instead of rebuilding.
type ListChangeKind int

const (
	// ListInsert: Count items were inserted starting at Index.
	ListInsert ListChangeKind = iota
	// ListRemove: Count items were removed starting at Index.
	ListRemove
	// ListReplace: the item at Index was replaced.
	ListReplace
	// ListMove: one item moved from Index to To.
	ListMove
	// ListReset: a wholesale change; re-read the whole list.
	ListReset
)

// ListEvent describes one change to an ObservableList.
type ListEvent[T any] struct {
	Kind  ListChangeKind
	Index int
	To    int // ListMove only: the destination index
	Count int // ListInsert / ListRemove
	Items []T // ListInsert / ListReplace: the affected items
}

// ObservableList is an ordered collection that emits a granular ListEvent on
// every mutation — the collection primitive of MVVM, bound to a ListBox /
// Table / TreeView. Out-of-range indices are clamped or ignored rather than
// panicking, matching the forgiving contract of a UI list.
type ObservableList[T any] struct {
	items  []T
	subs   map[int]func(ListEvent[T])
	nextID int
}

// NewObservableList builds a list seeded with the given items.
func NewObservableList[T any](initial ...T) *ObservableList[T] {
	return &ObservableList[T]{items: append([]T(nil), initial...)}
}

// Len returns the item count.
func (l *ObservableList[T]) Len() int { return len(l.items) }

// At returns the item at i (panics on out-of-range, like a slice index — At is
// a read used with a valid Len()-bounded index).
func (l *ObservableList[T]) At(i int) T { return l.items[i] }

// Slice returns a defensive copy of the items.
func (l *ObservableList[T]) Slice() []T { return append([]T(nil), l.items...) }

// Append adds items at the end and emits one ListInsert. Appending nothing is a
// no-op (no event).
func (l *ObservableList[T]) Append(v ...T) {
	if len(v) == 0 {
		return
	}
	idx := len(l.items)
	l.items = append(l.items, v...)
	l.emit(ListEvent[T]{Kind: ListInsert, Index: idx, Count: len(v), Items: append([]T(nil), v...)})
}

// Insert places v at index i (clamped to [0, Len]) and emits a ListInsert.
func (l *ObservableList[T]) Insert(i int, v T) {
	if i < 0 {
		i = 0
	}
	if i > len(l.items) {
		i = len(l.items)
	}
	l.items = append(l.items, v)
	copy(l.items[i+1:], l.items[i:])
	l.items[i] = v
	l.emit(ListEvent[T]{Kind: ListInsert, Index: i, Count: 1, Items: []T{v}})
}

// RemoveAt removes the item at i and emits a ListRemove. An out-of-range index
// is ignored (no event).
func (l *ObservableList[T]) RemoveAt(i int) {
	if i < 0 || i >= len(l.items) {
		return
	}
	l.items = append(l.items[:i], l.items[i+1:]...)
	l.emit(ListEvent[T]{Kind: ListRemove, Index: i, Count: 1})
}

// Set replaces the item at i and emits a ListReplace. An out-of-range index is
// ignored (no event).
func (l *ObservableList[T]) Set(i int, v T) {
	if i < 0 || i >= len(l.items) {
		return
	}
	l.items[i] = v
	l.emit(ListEvent[T]{Kind: ListReplace, Index: i, Items: []T{v}})
}

// Move relocates the item at from to index to and emits a ListMove. Either
// index out of range, or from == to, is ignored (no event).
func (l *ObservableList[T]) Move(from, to int) {
	n := len(l.items)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}
	v := l.items[from]
	l.items = append(l.items[:from], l.items[from+1:]...)
	l.items = append(l.items, v)
	copy(l.items[to+1:], l.items[to:])
	l.items[to] = v
	l.emit(ListEvent[T]{Kind: ListMove, Index: from, To: to})
}

// Clear removes every item and emits a ListReset.
func (l *ObservableList[T]) Clear() {
	l.items = nil
	l.emit(ListEvent[T]{Kind: ListReset})
}

// Subscribe registers fn (not called immediately) and returns an unsubscribe.
func (l *ObservableList[T]) Subscribe(fn func(ListEvent[T])) (unsubscribe func()) {
	if l.subs == nil {
		l.subs = map[int]func(ListEvent[T]){}
	}
	id := l.nextID
	l.nextID++
	l.subs[id] = fn
	return func() { delete(l.subs, id) }
}

// SubscribeChanged registers a value-less observer called on every change, so
// ObservableList satisfies Changeable (usable as a Command CanExecute source).
func (l *ObservableList[T]) SubscribeChanged(fn func()) (unsubscribe func()) {
	return l.Subscribe(func(ListEvent[T]) { fn() })
}

// emit fans an event out to every subscriber.
func (l *ObservableList[T]) emit(ev ListEvent[T]) {
	for _, fn := range l.subs {
		fn(ev)
	}
}
