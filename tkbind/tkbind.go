// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package tkbind holds the MVVM binding adapters that are specific to the pixel
// toolkit (github.com/go-widgets/toolkit) — the widgets whose value/callback
// shape the generic mvvm adapters can't express, such as a two-handle range
// slider. It is the ONLY MVVM package that imports toolkit; the core mvvm
// package stays backend-free.
package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// BindRange two-way-binds a RangeSlider's band to a pair of observables.
//
// The slider now OWNS its band as two mvvm.Observables rather than as settable
// fields plus a multi-argument OnChange, so this is two symmetric links rather
// than a callback fan-out — mvvm.BindTwoWay per handle.
//
// The band is normalised THROUGH the widget after the links exist, not before:
// Low().Set and High().Set do not clamp (only SetRange and the drag/key paths
// do), so a ViewModel holding an out-of-range or inverted band would otherwise
// seed an illegal slider. Calling SetRange once the links are live lets the
// widget's own invariant — clamped to [Min, Max], Low <= High — travel back to
// the observables, leaving both sides holding the same legal band.
//
// Loop-free by Observable.Set's equality check, as everywhere in this package.
// The returned unbind detaches all four subscriptions.
func BindRange(low, high *mvvm.Observable[float64], rs *toolkit.RangeSlider, invalidate func()) (unbind func()) {
	ul := mvvm.BindTwoWay(low, rs.Low(), invalidate)
	uh := mvvm.BindTwoWay(high, rs.High(), invalidate)
	rs.SetRange(low.Get(), high.Get())
	return func() {
		ul()
		uh()
	}
}
