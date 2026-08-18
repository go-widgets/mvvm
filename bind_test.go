// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

import "testing"

// fakeEntry mimics the field+callback surface of toolkit.Entry / tui.Entry
// without importing either backend — proving the adapters are backend-neutral.
type fakeEntry struct {
	Text     string
	OnChange func(string)
}

// fakeButton mimics a button's OnClick slot.
type fakeButton struct {
	OnClick func()
	Style   string // "default" / "disabled" — stands in for a real Style enum
}

func TestBindFieldTwoWay(t *testing.T) {
	obs := NewObservable("seed")
	e := &fakeEntry{}
	repaints := 0
	BindField(obs, &e.Text, &e.OnChange, func() { repaints++ })

	// Seeded from the observable.
	if e.Text != "seed" {
		t.Fatalf("seed: e.Text = %q", e.Text)
	}
	// View→VM: a user edit fires OnChange → observable updates.
	e.OnChange("typed")
	if obs.Get() != "typed" {
		t.Fatalf("view→vm: obs = %q, want typed", obs.Get())
	}
	// VM→View: setting the observable pushes into the field + repaints.
	obs.Set("pushed")
	if e.Text != "pushed" {
		t.Fatalf("vm→view: e.Text = %q, want pushed", e.Text)
	}
	// Two repaints: one from the user edit's echo push (obs.Set fires the
	// field-writer subscription) and one from the code-driven Set. The echo
	// write is idempotent; the extra invalidate coalesces to one frame.
	if repaints != 2 {
		t.Fatalf("repaints = %d, want 2 (edit echo + code push)", repaints)
	}
}

func TestBindFieldIsLoopFree(t *testing.T) {
	// The two-way echo must not recurse: a user edit sets the observable, whose
	// push writes the field silently — no re-entrant callback.
	obs := NewObservable(0)
	e := &fakeEntry{}
	var scale struct {
		Value    int
		OnChange func(int)
	}
	BindField(obs, &scale.Value, &scale.OnChange, nil) // nil invalidate branch
	fires := 0
	obs.Subscribe(func(int) { fires++ })
	scale.OnChange(5) // simulate a user drag
	if obs.Get() != 5 || scale.Value != 5 || fires != 1 {
		t.Fatalf("loop check: obs=%d field=%d fires=%d, want 5/5/1", obs.Get(), scale.Value, fires)
	}
	_ = e
}

func TestBindFieldComposesPriorCallbackAndUnbinds(t *testing.T) {
	obs := NewObservable("")
	e := &fakeEntry{}
	priorCalls := 0
	e.OnChange = func(string) { priorCalls++ } // a pre-existing handler
	unbind := BindField(obs, &e.Text, &e.OnChange, nil)

	e.OnChange("x") // both the prior handler AND the binding run
	if priorCalls != 1 || obs.Get() != "x" {
		t.Fatalf("compose: prior=%d obs=%q", priorCalls, obs.Get())
	}
	unbind() // restores the prior handler
	e.OnChange("y")
	if priorCalls != 2 {
		t.Fatalf("after unbind prior handler calls = %d, want 2", priorCalls)
	}
	if obs.Get() != "x" {
		t.Fatalf("after unbind the binding still updated obs: %q", obs.Get())
	}
}

func TestOneWay(t *testing.T) {
	obs := NewObservable(0.25)
	var bar struct{ Fraction float64 }
	repaints := 0
	unbind := OneWay(obs, &bar.Fraction, func() { repaints++ })
	if bar.Fraction != 0.25 {
		t.Fatalf("seed: %v", bar.Fraction)
	}
	obs.Set(0.75)
	if bar.Fraction != 0.75 || repaints != 1 {
		t.Fatalf("push: frac=%v repaints=%d", bar.Fraction, repaints)
	}
	unbind()
	obs.Set(1.0)
	if bar.Fraction != 0.75 {
		t.Fatalf("after unbind field changed: %v", bar.Fraction)
	}
	// nil-invalidate branch.
	var b2 struct{ V int }
	OneWay(NewObservable(3), &b2.V, nil)
	if b2.V != 3 {
		t.Fatalf("nil-invalidate seed: %d", b2.V)
	}
}

func TestBindCommandExecutesAndGreys(t *testing.T) {
	allow := false
	runs := 0
	c := NewCommand(func() { runs++ }, func() bool { return allow })
	b := &fakeButton{}
	prior := 0
	b.OnClick = func() { prior++ }
	unbind := BindCommand(c, &b.OnClick, func(ok bool) {
		if ok {
			b.Style = "default"
		} else {
			b.Style = "disabled"
		}
	})
	// setEnabled ran once with the initial (false) state.
	if b.Style != "disabled" {
		t.Fatalf("initial style = %q, want disabled", b.Style)
	}
	b.OnClick() // prior handler runs; command gated off → no run
	if prior != 1 || runs != 0 {
		t.Fatalf("gated click: prior=%d runs=%d", prior, runs)
	}
	allow = true
	c.RaiseCanExecuteChanged() // re-greys → enabled
	if b.Style != "default" {
		t.Fatalf("style after enable = %q", b.Style)
	}
	b.OnClick()
	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
	unbind()
	b.OnClick() // restored to prior-only
	if runs != 1 || prior != 3 {
		t.Fatalf("after unbind runs=%d prior=%d", runs, prior)
	}
}

func TestBindCommandNilSetEnabled(t *testing.T) {
	c := NewCommand(func() {}, nil)
	var onClick func()
	unbind := BindCommand(c, &onClick, nil) // nil setEnabled branch
	onClick()                               // must not panic
	unbind()
	if onClick != nil {
		t.Fatal("unbind should restore the (nil) prior handler")
	}
}

func TestBindListRebuildsAndProjects(t *testing.T) {
	l := NewObservableList(1, 2, 3)
	var items []string
	repaints := 0
	unbind := BindList(l, &items, func(n int) string {
		return string(rune('a' + n))
	}, func() { repaints++ })
	// Initial rebuild (1 repaint), projected b,c,d.
	if len(items) != 3 || items[0] != "b" || items[2] != "d" {
		t.Fatalf("initial items = %v", items)
	}
	if repaints != 1 {
		t.Fatalf("initial repaints = %d, want 1", repaints)
	}
	l.Append(25) // 'z'
	if len(items) != 4 || items[3] != "z" || repaints != 2 {
		t.Fatalf("after append items=%v repaints=%d", items, repaints)
	}
	unbind()
	l.Append(0)
	if len(items) != 4 {
		t.Fatalf("after unbind items changed: %v", items)
	}
	// nil-invalidate branch.
	var it2 []string
	BindList(NewObservableList("x"), &it2, func(s string) string { return s }, nil)
	if len(it2) != 1 || it2[0] != "x" {
		t.Fatalf("nil-invalidate list = %v", it2)
	}
}

func TestBindTwoWaySeedsFromSourceAndLinksBothWays(t *testing.T) {
	src := NewObservable(7)
	dst := NewObservable(0)
	repaints := 0
	unbind := BindTwoWay(src, dst, func() { repaints++ })

	if dst.Get() != 7 {
		t.Fatalf("seed: dst=%d, want 7 (src is the source of truth)", dst.Get())
	}
	src.Set(9)
	if dst.Get() != 9 {
		t.Fatalf("src→dst: dst=%d, want 9", dst.Get())
	}
	dst.Set(11)
	if src.Get() != 11 {
		t.Fatalf("dst→src: src=%d, want 11", src.Get())
	}
	if repaints == 0 {
		t.Fatal("changes should have requested repaints")
	}

	unbind()
	src.Set(1)
	if dst.Get() != 11 {
		t.Fatalf("after unbind src→dst still live: dst=%d", dst.Get())
	}
	dst.Set(2)
	if src.Get() != 1 {
		t.Fatalf("after unbind dst→src still live: src=%d", src.Get())
	}
}

// TestBindTwoWayIsLoopFree pins the property the whole package rests on: the
// echo must stop at the first hop rather than ping-pong. Counting notifications
// is what makes that visible — asserting the values agree would pass even if
// they had agreed after a thousand round trips.
func TestBindTwoWayIsLoopFree(t *testing.T) {
	src := NewObservable(0)
	dst := NewObservable(0)
	defer BindTwoWay(src, dst, nil)()

	srcNotes, dstNotes := 0, 0
	defer src.Subscribe(func(int) { srcNotes++ })()
	defer dst.Subscribe(func(int) { dstNotes++ })()

	src.Set(5)
	if srcNotes != 1 || dstNotes != 1 {
		t.Fatalf("one Set should notify each side once: src=%d dst=%d", srcNotes, dstNotes)
	}
	dst.Set(6)
	if srcNotes != 2 || dstNotes != 2 {
		t.Fatalf("the reverse Set should also settle at one: src=%d dst=%d", srcNotes, dstNotes)
	}
}

// TestBindTwoWayNilInvalidate covers the nil-invalidate branch on both
// directions.
func TestBindTwoWayNilInvalidate(t *testing.T) {
	src := NewObservable("a")
	dst := NewObservable("z")
	defer BindTwoWay(src, dst, nil)()
	src.Set("b")
	if dst.Get() != "b" {
		t.Fatalf("src→dst with nil invalidate: dst=%q", dst.Get())
	}
	dst.Set("c")
	if src.Get() != "c" {
		t.Fatalf("dst→src with nil invalidate: src=%q", src.Get())
	}
}
