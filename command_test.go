// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

import "testing"

func TestCommandExecuteGatedByCanExecute(t *testing.T) {
	allow := false
	runs := 0
	c := NewCommand(func() { runs++ }, func() bool { return allow })
	c.Execute() // CanExecute false → no run
	if runs != 0 {
		t.Fatalf("ran while disabled: %d", runs)
	}
	allow = true
	c.Execute()
	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
}

func TestCommandNilPredicateAlwaysExecutes(t *testing.T) {
	runs := 0
	c := NewCommand(func() { runs++ }, nil)
	if !c.CanExecute() {
		t.Fatal("nil predicate should be always executable")
	}
	c.Execute()
	if runs != 1 {
		t.Fatalf("runs = %d, want 1", runs)
	}
}

func TestCommandNilExecNoPanic(t *testing.T) {
	c := NewCommand(nil, nil)
	c.Execute() // must not panic
}

func TestCommandCanExecuteChangedSubscription(t *testing.T) {
	c := NewCommand(func() {}, func() bool { return true })
	fires := 0
	unsub := c.SubscribeCanExecuteChanged(func() { fires++ })
	c.RaiseCanExecuteChanged()
	unsub()
	c.RaiseCanExecuteChanged()
	if fires != 1 {
		t.Fatalf("CanExecuteChanged fires = %d, want 1", fires)
	}
}

func TestBindCanExecuteFromMixedSources(t *testing.T) {
	// A command whose CanExecute reads both a string and a bool observable —
	// heterogeneous sources combined through the Changeable interface.
	name := NewObservable("")
	agree := NewObservable(false)
	c := NewCommand(func() {}, func() bool { return name.Get() != "" && agree.Get() })
	raises := 0
	c.SubscribeCanExecuteChanged(func() { raises++ })
	unbind := BindCanExecute(c, name, agree)

	name.Set("Ada") // raises once
	agree.Set(true) // raises once
	if raises != 2 {
		t.Fatalf("raises = %d, want 2", raises)
	}
	if !c.CanExecute() {
		t.Fatal("command should now be executable")
	}
	unbind()
	name.Set("Bob") // detached → no raise
	if raises != 2 {
		t.Fatalf("after unbind raises = %d, want 2", raises)
	}
}
