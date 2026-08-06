// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// BrowserVM is the observable ViewModel for a toolkit.Browser: its navigable
// state exposed as Observables (for read-only binding to labels/indicators) and
// its actions as Commands (with Can… guards). The Browser widget itself stays a
// plain View in toolkit — it never imports mvvm — so this adapter is the MVVM
// seam: it mirrors the widget's state into the observables (via the widget's
// OnChange hook) and drives the widget from the commands.
type BrowserVM struct {
	URL        *mvvm.Observable[string]
	Title      *mvvm.Observable[string]
	Loading    *mvvm.Observable[bool]
	Progress   *mvvm.Observable[float64]
	CanBack    *mvvm.Observable[bool]
	CanForward *mvvm.Observable[bool]
	TabCount   *mvvm.Observable[int]
	Zoom       *mvvm.Observable[float64]
	Back       *mvvm.Command
	Forward    *mvvm.Command
	Reload     *mvvm.Command
	ZoomIn     *mvvm.Command
	ZoomOut    *mvvm.Command
}

// BindBrowser builds a BrowserVM bound to b and returns it with an unbind func.
// The widget is the source of truth (it owns the tabs/history); the VM mirrors
// its state on every change and its commands invoke the widget's actions. The
// commands' CanExecute tracks the widget's Can… guards, and their enabled state
// is refreshed on every widget change. invalidate (optional) is called after
// each sync so a host can schedule a redraw.
func BindBrowser(b *toolkit.Browser, invalidate func()) (*BrowserVM, func()) {
	vm := &BrowserVM{
		URL:        mvvm.NewObservable(""),
		Title:      mvvm.NewObservable(""),
		Loading:    mvvm.NewObservable(false),
		Progress:   mvvm.NewObservable(0.0),
		CanBack:    mvvm.NewObservable(false),
		CanForward: mvvm.NewObservable(false),
		TabCount:   mvvm.NewObservable(0),
		Zoom:       mvvm.NewObservable(0.0),
	}
	vm.Back = mvvm.NewCommand(b.Back, b.CanBack)
	vm.Forward = mvvm.NewCommand(b.Forward, b.CanForward)
	vm.Reload = mvvm.NewCommand(b.Reload, func() bool { return b.CurrentURL() != "" })
	vm.ZoomIn = mvvm.NewCommand(b.ZoomIn, b.CanZoomIn)
	vm.ZoomOut = mvvm.NewCommand(b.ZoomOut, b.CanZoomOut)

	sync := func() {
		vm.URL.Set(b.CurrentURL())
		vm.Title.Set(b.ActiveTitle())
		vm.Loading.Set(b.Loading())
		vm.Progress.Set(b.Progress())
		vm.CanBack.Set(b.CanBack())
		vm.CanForward.Set(b.CanForward())
		vm.TabCount.Set(b.TabCount())
		vm.Zoom.Set(b.Zoom())
		vm.Back.RaiseCanExecuteChanged()
		vm.Forward.RaiseCanExecuteChanged()
		vm.Reload.RaiseCanExecuteChanged()
		vm.ZoomIn.RaiseCanExecuteChanged()
		vm.ZoomOut.RaiseCanExecuteChanged()
		if invalidate != nil {
			invalidate()
		}
	}
	prev := b.OnChange
	b.OnChange = func() {
		if prev != nil {
			prev()
		}
		sync()
	}
	sync() // seed the observables from the widget's current state
	return vm, func() { b.OnChange = prev }
}
