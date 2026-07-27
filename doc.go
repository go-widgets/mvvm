// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package mvvm is a tiny, dependency-free MVVM (Model-View-ViewModel) layer for
// the go-widgets ecosystem. It provides the three MVVM primitives —
// [Observable] (a bindable property), [Command] (a bindable action), and
// [ObservableList] (a bindable collection) — plus pointer-based binding
// adapters ([BindField], [OneWay], [BindCommand], [BindList]) that wire those
// primitives to widgets.
//
// # Backend-agnostic by construction
//
// This package imports NEITHER go-widgets/toolkit (pixel widgets) nor
// go-widgets/tui (terminal-cell widgets). The adapters reach a widget only
// through a pointer to its value field and a pointer to its callback slot, e.g.
// (&entry.Text, &entry.OnChange). Because the two backends mirror the same
// field names and callback signatures for each widget, a single ViewModel and a
// single set of bindings drive both. Multi-argument or hook-less widgets (a
// two-handle range slider, a plain viewer table) get small bespoke adapters in
// per-backend subpackages that import their backend; the generic core here does
// not.
//
// # Why binding is loop-free
//
// A widget fires its change callback from inside its event handler AND leaves
// direct field writes silent. So the View→ViewModel edge is the callback, the
// ViewModel→View edge is a silent field write, and [Observable.Set] skips equal
// values — a two-way binding can echo without recursing.
//
// # Threading
//
// The primitives are not safe for concurrent use. Mutate observables on the UI
// goroutine; for an async producer, hand the value to the app's refresh queue
// and Set it from the UI tick.
//
// # A form in ~10 lines
//
//	type FormVM struct {
//		Name  *mvvm.Observable[string]
//		Names *mvvm.ObservableList[string]
//		Save  *mvvm.Command
//	}
//
//	vm := &FormVM{Name: mvvm.NewObservable(""), Names: mvvm.NewObservableList[string]()}
//	vm.Save = mvvm.NewCommand(
//		func() { vm.Names.Append(vm.Name.Get()); vm.Name.Set("") },
//		func() bool { return vm.Name.Get() != "" }, // CanExecute
//	)
//	mvvm.BindCanExecute(vm.Save, vm.Name)
//
//	// View (pixel): identical for tui except the widget types + repaint hook.
//	mvvm.BindField(vm.Name, &name.Text, &name.OnChange, repaint)
//	mvvm.BindCommand(vm.Save, &save.OnClick, setEnabled)
//	mvvm.BindList(vm.Names, &list.Items, func(s string) string { return s }, repaint)
package mvvm
