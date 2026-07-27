// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package tuibind holds the MVVM binding adapters specific to the terminal-cell
// toolkit (github.com/go-widgets/tui) — widgets whose callback shape the generic
// mvvm adapters can't express, such as a Dropdown whose OnChange carries both
// the index and the value string. It is the ONLY MVVM package that imports tui;
// the core mvvm package stays backend-free.
package tuibind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/tui"
)

// BindDropdown two-way-binds a Dropdown's Selected index to an observable. The
// Dropdown reports OnChange(idx, value) — a multi-argument shape the generic
// BindField can't take — so this adapter keeps the observable in sync with the
// index and pushes the index back. Loop-free (silent field write + equal-skip).
// Returns an unbind restoring the prior OnChange and detaching.
func BindDropdown(sel *mvvm.Observable[int], dd *tui.Dropdown, invalidate func()) (unbind func()) {
	dd.Selected = sel.Get()
	prev := dd.OnChange
	dd.OnChange = func(idx int, value string) {
		if prev != nil {
			prev(idx, value)
		}
		sel.Set(idx)
	}
	unsub := sel.Subscribe(func(v int) {
		dd.Selected = v
		if invalidate != nil {
			invalidate()
		}
	})
	return func() {
		dd.OnChange = prev
		unsub()
	}
}

// BindTableSelection two-way-binds a Table's Selected row to an observable. The
// Table's hook is OnSelect(row) (a distinct name from the generic OnChange), so
// it needs its own adapter. Returns an unbind restoring the prior OnSelect and
// detaching.
func BindTableSelection(sel *mvvm.Observable[int], tb *tui.Table, invalidate func()) (unbind func()) {
	tb.Selected = sel.Get()
	prev := tb.OnSelect
	tb.OnSelect = func(row int) {
		if prev != nil {
			prev(row)
		}
		sel.Set(row)
	}
	unsub := sel.Subscribe(func(v int) {
		tb.Selected = v
		if invalidate != nil {
			invalidate()
		}
	})
	return func() {
		tb.OnSelect = prev
		unsub()
	}
}
