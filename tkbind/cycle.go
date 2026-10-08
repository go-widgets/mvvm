// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// BindCycle two-way-binds a cycle button's position to an observable index,
// clamped to the options it has.
//
// Clamped in the OBSERVABLE before the link exists, for the reason given on
// the clamping binders in controls.go: BindTwoWay seeds the destination by
// writing its Observable, which is not the path any of the widget's own
// guards sit on.
func BindCycle(index *mvvm.Observable[int], c *toolkit.CycleButton, invalidate func()) (unbind func()) {
	if n := len(c.Options); n > 0 {
		if i := index.Get(); i < 0 {
			index.Set(0)
		} else if i >= n {
			index.Set(n - 1)
		}
	}
	return mvvm.BindTwoWay(index, c.Index(), invalidate)
}

// BindCycleValues binds a cycle button to an observable of what the positions
// MEAN, through a table of one value per option.
//
// ⛔ This is the binder a cycle button actually wants, and its absence is why
// apps wire one by hand. A cycle button holds a POSITION; a ViewModel holds a
// value — a quarter turn of 90, 180 or 270 against a button sitting at 0, 1 or
// 2 — and the two are not the same datum, so there is nothing for BindCycle to
// keep equal. What looks like "this one cannot be bound" is a mapping that was
// never written down, and the app ends up with
//
//	c.Index().Subscribe(func(i int) { vm.Turn = 90 * (i + 1) })
//
// which is one direction, in the consumer, with the mapping spelled as
// arithmetic that only reads correctly forwards.
//
// values[i] is what position i means. A value the table does not hold settles
// on the first position and is written back, so the two sides never disagree
// about what the button is showing; a nil or empty table binds nothing and
// returns a no-op, because a mapping with no entries cannot say what any
// position means.
func BindCycleValues[T comparable](value *mvvm.Observable[T], c *toolkit.CycleButton, values []T, invalidate func()) (unbind func()) {
	if len(values) == 0 {
		return func() {}
	}
	at := func(v T) int {
		for i, x := range values {
			if x == v {
				return i
			}
		}
		return 0
	}

	// Settle both sides on a position the table has, before either can tell
	// the other about a value it cannot represent.
	i := at(value.Get())
	value.Set(values[i])
	c.Index().Set(i)

	unIdx := c.Index().Subscribe(func(i int) {
		if i >= 0 && i < len(values) {
			value.Set(values[i])
		}
		if invalidate != nil {
			invalidate()
		}
	})
	unVal := value.Subscribe(func(v T) {
		c.Index().Set(at(v))
		if invalidate != nil {
			invalidate()
		}
	})
	return func() {
		unIdx()
		unVal()
	}
}

// BindButton wires a button to a command: pressing it executes, and the
// command's CanExecute drives whether the button can be pressed at all.
//
// ⛔ The enabling is the half that is missing when an app passes a plain func
// to NewButton. A handler that checks its own preconditions and returns having
// written a sentence into a status line is a control that looks pressable and
// is not, and every one of those checks is a copy of a rule the rest of the
// program already knows. A Command carries the rule once, and the button goes
// grey by itself.
func BindButton(cmd *mvvm.Command, b *toolkit.Button, invalidate func()) (unbind func()) {
	return mvvm.BindCommand(cmd, &b.OnClick, func(on bool) {
		b.Disabled().Set(!on)
		if invalidate != nil {
			invalidate()
		}
	})
}
