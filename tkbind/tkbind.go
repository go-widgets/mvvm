// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package tkbind holds the MVVM binding adapters that are specific to the pixel
// toolkit (github.com/go-widgets/toolkit) — the widgets whose value/callback
// shape the generic mvvm adapters can't express, such as a two-handle range
// slider (a multi-argument OnChange). It is the ONLY MVVM package that imports
// toolkit; the core mvvm package stays backend-free.
package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// BindRange two-way-binds a RangeSlider's Low/High to a pair of observables.
// The slider fires OnChange(low, high) on every drag — a shape the generic
// single-value BindField can't take — so this adapter fans it out to both
// observables and pushes each back to the matching field. Loop-free: the field
// writes are silent and Observable.Set skips equal values. Returns an unbind
// that restores the prior OnChange and detaches both subscriptions.
func BindRange(low, high *mvvm.Observable[float64], rs *toolkit.RangeSlider, invalidate func()) (unbind func()) {
	rs.Low = low.Get()
	rs.High = high.Get()
	prev := rs.OnChange
	rs.OnChange = func(lo, hi float64) {
		if prev != nil {
			prev(lo, hi)
		}
		low.Set(lo)
		high.Set(hi)
	}
	ul := low.Subscribe(func(v float64) {
		rs.Low = v
		if invalidate != nil {
			invalidate()
		}
	})
	uh := high.Subscribe(func(v float64) {
		rs.High = v
		if invalidate != nil {
			invalidate()
		}
	})
	return func() {
		rs.OnChange = prev
		ul()
		uh()
	}
}
