// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

import "testing"

func TestObservableGetSetNotify(t *testing.T) {
	o := NewObservable("a")
	if o.Get() != "a" {
		t.Fatalf("Get = %q, want a", o.Get())
	}
	var seen []string
	o.Subscribe(func(v string) { seen = append(seen, v) })
	o.Set("b")
	o.Set("c")
	if len(seen) != 2 || seen[0] != "b" || seen[1] != "c" {
		t.Fatalf("notifications = %v, want [b c]", seen)
	}
	if o.Get() != "c" {
		t.Fatalf("Get = %q, want c", o.Get())
	}
}

func TestObservableSkipsEqual(t *testing.T) {
	o := NewObservable(7)
	fires := 0
	o.Subscribe(func(int) { fires++ })
	o.Set(7) // equal → no notify
	if fires != 0 {
		t.Fatalf("equal Set fired %d times, want 0", fires)
	}
	o.Set(8)
	if fires != 1 {
		t.Fatalf("changed Set fired %d times, want 1", fires)
	}
}

func TestObservableEqNilAlwaysNotifies(t *testing.T) {
	// A nil eq means "never equal": every Set notifies, even with the same value.
	o := NewObservableEq([]int{1}, nil)
	fires := 0
	o.Subscribe(func([]int) { fires++ })
	o.Set([]int{1})
	o.Set([]int{1})
	if fires != 2 {
		t.Fatalf("nil-eq Set fired %d times, want 2", fires)
	}
}

func TestObservableCustomEq(t *testing.T) {
	// Length-based equality: same length ⇒ no notify.
	eq := func(a, b []int) bool { return len(a) == len(b) }
	o := NewObservableEq([]int{1, 2}, eq)
	fires := 0
	o.Subscribe(func([]int) { fires++ })
	o.Set([]int{9, 9}) // same length → skipped
	if fires != 0 {
		t.Fatalf("same-length Set fired %d, want 0", fires)
	}
	o.Set([]int{1}) // different length → notify
	if fires != 1 {
		t.Fatalf("diff-length Set fired %d, want 1", fires)
	}
}

func TestObservableUnsubscribe(t *testing.T) {
	o := NewObservable(0)
	fires := 0
	unsub := o.Subscribe(func(int) { fires++ })
	o.Set(1)
	unsub()
	o.Set(2)
	if fires != 1 {
		t.Fatalf("after unsubscribe fires = %d, want 1", fires)
	}
}

func TestObservableReentrantSetConverges(t *testing.T) {
	// A subscriber that normalises the value by calling Set again must converge,
	// not recurse: negatives are clamped to 0 in a single settled pass.
	o := NewObservable(0)
	var seen []int
	o.Subscribe(func(v int) {
		seen = append(seen, v)
		if v < 0 {
			o.Set(0) // re-entrant normalisation
		}
	})
	o.Set(-5)
	if o.Get() != 0 {
		t.Fatalf("Get = %d, want 0 (normalised)", o.Get())
	}
	// Two passes: first with -5, second (dirty) with 0; the 0-pass sets nothing.
	if len(seen) != 2 || seen[0] != -5 || seen[1] != 0 {
		t.Fatalf("passes = %v, want [-5 0]", seen)
	}
}

func TestObservableSubscribeChangedAndChangeable(t *testing.T) {
	o := NewObservable("x")
	fires := 0
	unsub := o.SubscribeChanged(func() { fires++ })
	// Usable through the Changeable interface.
	var _ Changeable = o
	o.Set("y")
	unsub()
	o.Set("z")
	if fires != 1 {
		t.Fatalf("SubscribeChanged fires = %d, want 1", fires)
	}
}
