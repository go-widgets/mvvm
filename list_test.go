// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

import "testing"

// record captures every event a list emits, for assertions.
func record[T any](l *ObservableList[T]) *[]ListEvent[T] {
	evs := &[]ListEvent[T]{}
	l.Subscribe(func(e ListEvent[T]) { *evs = append(*evs, e) })
	return evs
}

func TestListAppendAndInsert(t *testing.T) {
	l := NewObservableList("a")
	evs := record(l)
	l.Append("b", "c")
	l.Append()        // empty append → no event
	l.Insert(1, "x")  // a x b c
	l.Insert(-3, "L") // clamp to front: L a x b c
	l.Insert(99, "R") // clamp to end:  L a x b c R
	got := l.Slice()
	want := []string{"L", "a", "x", "b", "c", "R"}
	if len(got) != len(want) {
		t.Fatalf("items = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("items[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// One insert (b,c), one insert(x), one insert(L), one insert(R) = 4 events.
	if len(*evs) != 4 {
		t.Fatalf("events = %d, want 4", len(*evs))
	}
	if (*evs)[0].Kind != ListInsert || (*evs)[0].Count != 2 || (*evs)[0].Index != 1 {
		t.Fatalf("first event = %+v", (*evs)[0])
	}
}

func TestListRemoveAndBounds(t *testing.T) {
	l := NewObservableList(1, 2, 3)
	evs := record(l)
	l.RemoveAt(1)  // 1 3
	l.RemoveAt(-1) // ignored
	l.RemoveAt(50) // ignored
	if l.Len() != 2 || l.At(0) != 1 || l.At(1) != 3 {
		t.Fatalf("after remove = %v", l.Slice())
	}
	if len(*evs) != 1 || (*evs)[0].Kind != ListRemove || (*evs)[0].Index != 1 {
		t.Fatalf("events = %+v, want one ListRemove@1", *evs)
	}
}

func TestListSetAndBounds(t *testing.T) {
	l := NewObservableList("a", "b")
	evs := record(l)
	l.Set(1, "B")  // a B
	l.Set(9, "Z")  // ignored
	l.Set(-1, "Z") // ignored
	if l.At(1) != "B" {
		t.Fatalf("Set failed: %v", l.Slice())
	}
	if len(*evs) != 1 || (*evs)[0].Kind != ListReplace || (*evs)[0].Items[0] != "B" {
		t.Fatalf("events = %+v", *evs)
	}
}

func TestListMove(t *testing.T) {
	l := NewObservableList("a", "b", "c", "d")
	evs := record(l)
	l.Move(0, 2)  // b c a d
	l.Move(1, 1)  // no-op (from==to)
	l.Move(-1, 2) // ignored
	l.Move(0, 9)  // ignored
	got := l.Slice()
	want := []string{"b", "c", "a", "d"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("after move = %v, want %v", got, want)
		}
	}
	if len(*evs) != 1 || (*evs)[0].Kind != ListMove || (*evs)[0].Index != 0 || (*evs)[0].To != 2 {
		t.Fatalf("events = %+v", *evs)
	}
}

func TestListClearAndReset(t *testing.T) {
	l := NewObservableList(1, 2, 3)
	evs := record(l)
	l.Clear()
	if l.Len() != 0 {
		t.Fatalf("Clear left %d items", l.Len())
	}
	if len(*evs) != 1 || (*evs)[0].Kind != ListReset {
		t.Fatalf("events = %+v, want one ListReset", *evs)
	}
}

func TestListUnsubscribeAndChangeable(t *testing.T) {
	l := NewObservableList[int]()
	fires := 0
	unsub := l.SubscribeChanged(func() { fires++ })
	var _ Changeable = l
	l.Append(1)
	unsub()
	l.Append(2)
	if fires != 1 {
		t.Fatalf("SubscribeChanged fires = %d, want 1", fires)
	}
}
