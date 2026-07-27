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
	priorCalls := 0
	rs.OnChange = func(lo, hi float64) { priorCalls++ } // pre-existing handler
	repaints := 0
	unbind := BindRange(low, high, rs, func() { repaints++ })

	// Seeded from the observables.
	if rs.Low != 10 || rs.High != 90 {
		t.Fatalf("seed: Low=%v High=%v, want 10/90", rs.Low, rs.High)
	}
	// View→VM: a drag fires OnChange(lo,hi) → both observables update, and the
	// prior handler still runs.
	rs.OnChange(20, 80)
	if low.Get() != 20 || high.Get() != 80 || priorCalls != 1 {
		t.Fatalf("view→vm: low=%v high=%v prior=%d", low.Get(), high.Get(), priorCalls)
	}
	// VM→View: setting an observable pushes to the matching field.
	low.Set(5)
	high.Set(95)
	if rs.Low != 5 || rs.High != 95 {
		t.Fatalf("vm→view: Low=%v High=%v, want 5/95", rs.Low, rs.High)
	}
	if repaints == 0 {
		t.Fatal("VM→View pushes should have requested repaints")
	}
	// Unbind restores the prior handler and detaches.
	unbind()
	low.Set(0)
	if rs.Low != 5 {
		t.Fatalf("after unbind Low changed to %v", rs.Low)
	}
	rs.OnChange(1, 2) // only the prior handler now
	if priorCalls != 2 || low.Get() != 0 && high.Get() != 95 {
		t.Fatalf("after unbind: prior=%d low=%v high=%v", priorCalls, low.Get(), high.Get())
	}
}

func TestBindRangeNilPriorAndInvalidate(t *testing.T) {
	// Covers the prev==nil and invalidate==nil branches.
	low := mvvm.NewObservable(0.0)
	high := mvvm.NewObservable(100.0)
	rs := toolkit.NewRangeSlider(0, 100, 50, 50)
	unbind := BindRange(low, high, rs, nil)
	defer unbind()
	rs.OnChange(10, 90) // prev is nil — must not panic
	if low.Get() != 10 || high.Get() != 90 {
		t.Fatalf("nil-prior view→vm: low=%v high=%v", low.Get(), high.Get())
	}
	low.Set(25) // nil invalidate — must not panic
	if rs.Low != 25 {
		t.Fatalf("nil-invalidate push: Low=%v", rs.Low)
	}
}
