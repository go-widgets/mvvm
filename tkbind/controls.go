// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// The four everyday input controls, bound the way BindRange binds a slider:
// one mvvm.BindTwoWay per reactive datum, and where the widget holds an
// invariant the value is normalised THROUGH THE WIDGET once the link is live,
// so an illegal value in the ViewModel cannot seed an illegal control.
//
// ⛔ They exist because without them an app binds these by hand, and binding by
// hand is not binding. The shape it ends up with is
//
//	sp.Value().Subscribe(func(v int) { vm.Count = v })   // a copy out
//	... and a rebuild of the widget to get the value back in
//
// which is a plain field copied across a boundary plus an imperative resync --
// the thing MVVM is for not having. The one-way direction is easy to write and
// it is the one that hides the problem, because it works until something else
// changes the model.
//
// Each returns an unbind that detaches every subscription it made.

// BindEntry two-way-binds a text box to an observable string.
//
// No normalisation: an Entry accepts any string, so there is no invariant to
// travel back and the ViewModel's value is simply shown.
func BindEntry(text *mvvm.Observable[string], e *toolkit.Entry, invalidate func()) (unbind func()) {
	return mvvm.BindTwoWay(text, e.Text(), invalidate)
}

// ⛔ A note that applies to both of the clamping binders below.
//
// BindTwoWay seeds the destination by writing its Observable, which goes ROUND
// whatever guard the widget's own setter holds -- SpinButton.SetValue clamps
// to [Min, Max] and DropDown.Select refuses an index it has no option for, and
// neither of them is on the path BindTwoWay takes. So normalising afterwards
// does not work either: by then the widget is already holding the illegal
// value, and asking Select to fix it is asking the guard that just refused.
//
// The value is therefore made legal in the OBSERVABLE, before the link exists.

// BindSpin two-way-binds a spin button to an observable int, clamped to the
// control's range.
func BindSpin(value *mvvm.Observable[int], sp *toolkit.SpinButton, invalidate func()) (unbind func()) {
	if v := value.Get(); v < sp.Min {
		value.Set(sp.Min)
	} else if v > sp.Max {
		value.Set(sp.Max)
	}
	return mvvm.BindTwoWay(value, sp.Value(), invalidate)
}

// BindChoice two-way-binds a drop-down's selection to an observable index,
// clamped to the options it actually has.
func BindChoice(index *mvvm.Observable[int], d *toolkit.DropDown, invalidate func()) (unbind func()) {
	if n := len(d.Options); n > 0 {
		if i := index.Get(); i < 0 {
			index.Set(0)
		} else if i >= n {
			index.Set(n - 1)
		}
	}
	return mvvm.BindTwoWay(index, d.Selected(), invalidate)
}

// BindCheck two-way-binds a tick box to an observable bool.
func BindCheck(on *mvvm.Observable[bool], c *toolkit.CheckButton, invalidate func()) (unbind func()) {
	return mvvm.BindTwoWay(on, c.Checked(), invalidate)
}
