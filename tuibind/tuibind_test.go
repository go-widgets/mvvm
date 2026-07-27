// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tuibind

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/tui"
)

func TestBindDropdownTwoWay(t *testing.T) {
	sel := mvvm.NewObservable(1)
	dd := tui.NewDropdown([]string{"a", "b", "c"}, 0)
	priorCalls := 0
	dd.OnChange = func(idx int, value string) { priorCalls++ }
	repaints := 0
	unbind := BindDropdown(sel, dd, func() { repaints++ })

	if dd.Selected != 1 {
		t.Fatalf("seed: Selected=%d, want 1", dd.Selected)
	}
	dd.OnChange(2, "c") // View→VM
	if sel.Get() != 2 || priorCalls != 1 {
		t.Fatalf("view→vm: sel=%d prior=%d", sel.Get(), priorCalls)
	}
	sel.Set(0) // VM→View
	if dd.Selected != 0 || repaints == 0 {
		t.Fatalf("vm→view: Selected=%d repaints=%d", dd.Selected, repaints)
	}
	unbind()
	sel.Set(2)
	if dd.Selected != 0 {
		t.Fatalf("after unbind Selected changed to %d", dd.Selected)
	}
}

func TestBindDropdownNilPriorAndInvalidate(t *testing.T) {
	sel := mvvm.NewObservable(0)
	dd := tui.NewDropdown([]string{"x", "y"}, 0)
	unbind := BindDropdown(sel, dd, nil)
	defer unbind()
	dd.OnChange(1, "y") // nil prior — no panic
	if sel.Get() != 1 {
		t.Fatalf("nil-prior view→vm: sel=%d", sel.Get())
	}
	sel.Set(0) // nil invalidate — no panic
	if dd.Selected != 0 {
		t.Fatalf("nil-invalidate push: Selected=%d", dd.Selected)
	}
}

func TestBindTableSelectionTwoWay(t *testing.T) {
	sel := mvvm.NewObservable(2)
	tb := tui.NewTable(
		[]tui.TableColumn{{Title: "Name"}},
		[][]string{{"a"}, {"b"}, {"c"}, {"d"}},
	)
	priorCalls := 0
	tb.OnSelect = func(row int) { priorCalls++ }
	repaints := 0
	unbind := BindTableSelection(sel, tb, func() { repaints++ })

	if tb.Selected != 2 {
		t.Fatalf("seed: Selected=%d, want 2", tb.Selected)
	}
	tb.OnSelect(1) // View→VM
	if sel.Get() != 1 || priorCalls != 1 {
		t.Fatalf("view→vm: sel=%d prior=%d", sel.Get(), priorCalls)
	}
	sel.Set(3) // VM→View
	if tb.Selected != 3 || repaints == 0 {
		t.Fatalf("vm→view: Selected=%d repaints=%d", tb.Selected, repaints)
	}
	unbind()
	sel.Set(0)
	if tb.Selected != 3 {
		t.Fatalf("after unbind Selected changed to %d", tb.Selected)
	}
}

func TestBindTableSelectionNilPriorAndInvalidate(t *testing.T) {
	sel := mvvm.NewObservable(0)
	tb := tui.NewTable([]tui.TableColumn{{Title: "C"}}, [][]string{{"a"}, {"b"}})
	unbind := BindTableSelection(sel, tb, nil)
	defer unbind()
	tb.OnSelect(1) // nil prior — no panic
	if sel.Get() != 1 {
		t.Fatalf("nil-prior view→vm: sel=%d", sel.Get())
	}
	sel.Set(0) // nil invalidate — no panic
	if tb.Selected != 0 {
		t.Fatalf("nil-invalidate push: Selected=%d", tb.Selected)
	}
}
