// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"testing"

	"github.com/go-widgets/toolkit"
)

func TestBindBrowser(t *testing.T) {
	b := toolkit.NewBrowser()
	navd := 0
	b.OnNavigate = func(target string, width int) { navd++ } // host render seam (no-op)
	priorCalls := 0
	b.OnChange = func() { priorCalls++ } // pre-existing hook must be preserved
	repaints := 0

	vm, unbind := BindBrowser(b, func() { repaints++ })

	// Seeded from the widget's (empty) state.
	if vm.URL.Get() != "" || vm.TabCount.Get() != 0 || vm.Loading.Get() {
		t.Fatalf("seed: url=%q tabs=%d loading=%v", vm.URL.Get(), vm.TabCount.Get(), vm.Loading.Get())
	}
	if vm.CanBack.Get() || vm.CanForward.Get() {
		t.Fatal("fresh browser should allow neither back nor forward")
	}
	if vm.Back.CanExecute() || vm.Forward.CanExecute() || vm.Reload.CanExecute() {
		t.Fatal("no page yet → all nav commands disabled")
	}
	seedRepaints := repaints // the seed sync calls invalidate once

	// Opening a page mirrors into the VM, preserves the prior OnChange, and
	// requests a repaint.
	b.Open("https://a/", "A")
	if vm.URL.Get() != "https://a/" || !vm.Loading.Get() || vm.TabCount.Get() != 1 {
		t.Fatalf("after Open: url=%q loading=%v tabs=%d", vm.URL.Get(), vm.Loading.Get(), vm.TabCount.Get())
	}
	if priorCalls == 0 {
		t.Fatal("prior OnChange hook should still fire")
	}
	if repaints == seedRepaints {
		t.Fatal("Open should have requested a repaint via invalidate")
	}
	if navd == 0 {
		t.Fatal("Open should trigger the host navigate seam")
	}
	if !vm.Reload.CanExecute() {
		t.Fatal("a current page → Reload enabled")
	}

	// Deliver a page + navigate onward so Back becomes possible, then the Back
	// command (VM→View) moves the widget and the guard tracks it.
	b.Deliver("https://a/", nil, 0, 0, 400, nil, "A")
	b.Navigate("https://a/b")
	if !vm.CanBack.Get() || !vm.Back.CanExecute() {
		t.Fatalf("after navigate: canBack obs=%v cmd=%v", vm.CanBack.Get(), vm.Back.CanExecute())
	}
	vm.Back.Execute() // command drives the widget back
	if vm.CanForward.Get() != b.CanForward() || b.CurrentURL() != "https://a/" {
		t.Fatalf("after Back cmd: url=%q canFwd=%v", b.CurrentURL(), vm.CanForward.Get())
	}
	vm.Forward.Execute()
	if b.CurrentURL() != "https://a/b" {
		t.Fatalf("after Forward cmd: url=%q, want https://a/b", b.CurrentURL())
	}
	vm.Reload.Execute() // exercise the reload command path

	// Progress mirrors through.
	b.SetProgress(0.5)
	if vm.Progress.Get() != 0.5 {
		t.Fatalf("progress = %v, want 0.5", vm.Progress.Get())
	}

	// Unbind restores the prior hook; further widget changes no longer sync.
	unbind()
	before := vm.URL.Get()
	pc := priorCalls
	b.Open("https://z/", "Z")
	if vm.URL.Get() != before {
		t.Fatal("after unbind the VM should not track the widget")
	}
	if priorCalls == pc {
		t.Fatal("after unbind the prior OnChange hook should be restored + firing")
	}
}

// TestBindBrowserNilInvalidate covers the nil-invalidate branch (no repaint
// callback supplied).
func TestBindBrowserNilInvalidate(t *testing.T) {
	b := toolkit.NewBrowser()
	vm, unbind := BindBrowser(b, nil)
	defer unbind()
	b.Open("https://x/", "X")
	if vm.URL.Get() != "https://x/" {
		t.Fatalf("url = %q, want https://x/", vm.URL.Get())
	}
}
