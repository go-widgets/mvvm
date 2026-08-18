// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

func TestBindRangeTwoWay(t *testing.T) {
	low := mvvm.NewObservable(10.0)
	high := mvvm.NewObservable(90.0)
	rs := toolkit.NewRangeSlider(0, 100, 0, 0)
	repaints := 0
	unbind := BindRange(low, high, rs, func() { repaints++ })

	// Seeded from the observables: the ViewModel wins over the band the slider
	// was constructed with.
	if rs.Low().Get() != 10 || rs.High().Get() != 90 {
		t.Fatalf("seed: Low=%v High=%v, want 10/90", rs.Low().Get(), rs.High().Get())
	}
	// View→VM: the widget Sets its own observables on a drag or key press.
	rs.Low().Set(20)
	rs.High().Set(80)
	if low.Get() != 20 || high.Get() != 80 {
		t.Fatalf("view→vm: low=%v high=%v, want 20/80", low.Get(), high.Get())
	}
	// VM→View.
	low.Set(5)
	high.Set(95)
	if rs.Low().Get() != 5 || rs.High().Get() != 95 {
		t.Fatalf("vm→view: Low=%v High=%v, want 5/95", rs.Low().Get(), rs.High().Get())
	}
	if repaints == 0 {
		t.Fatal("VM→View pushes should have requested repaints")
	}
	// Unbind detaches BOTH directions.
	unbind()
	low.Set(0)
	if rs.Low().Get() != 5 {
		t.Fatalf("after unbind VM→View still live: Low=%v", rs.Low().Get())
	}
	rs.High().Set(42)
	if high.Get() != 95 {
		t.Fatalf("after unbind View→VM still live: high=%v", high.Get())
	}
}

// TestBindRangeNormalisesThroughTheWidget pins the reason SetRange is called
// after the links exist rather than before: Low().Set and High().Set do not
// clamp, so an out-of-range, inverted ViewModel band must be corrected by the
// widget and the correction must travel BACK, leaving both sides equal and
// legal.
func TestBindRangeNormalisesThroughTheWidget(t *testing.T) {
	low := mvvm.NewObservable(140.0) // above Max, and above high
	high := mvvm.NewObservable(-20.0)
	rs := toolkit.NewRangeSlider(0, 100, 10, 90)
	defer BindRange(low, high, rs, nil)()

	if rs.Low().Get() != 0 || rs.High().Get() != 100 {
		t.Fatalf("widget band not normalised: Low=%v High=%v, want 0/100",
			rs.Low().Get(), rs.High().Get())
	}
	if low.Get() != 0 || high.Get() != 100 {
		t.Fatalf("correction did not travel back: low=%v high=%v, want 0/100",
			low.Get(), high.Get())
	}
}

// TestBindRangeNilInvalidate covers the nil-invalidate branch.
func TestBindRangeNilInvalidate(t *testing.T) {
	low := mvvm.NewObservable(0.0)
	high := mvvm.NewObservable(100.0)
	rs := toolkit.NewRangeSlider(0, 100, 50, 50)
	defer BindRange(low, high, rs, nil)()
	low.Set(25) // nil invalidate — must not panic
	if rs.Low().Get() != 25 {
		t.Fatalf("nil-invalidate push: Low=%v", rs.Low().Get())
	}
	rs.High().Set(75)
	if high.Get() != 75 {
		t.Fatalf("nil-invalidate view→vm: high=%v", high.Get())
	}
}
