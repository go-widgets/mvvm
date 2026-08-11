// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package undo_test

import (
	"fmt"
	"testing"

	"github.com/go-widgets/mvvm/undo"
)

// counter is a tiny model whose mutations are recorded as commands, letting the
// tests assert the EXACT restored state after undo/redo (not merely "changed").
type counter struct{ n int }

func (c *counter) add(d int) undo.Command {
	return undo.NewCommand(fmt.Sprintf("Add %d", d),
		func() { c.n += d },
		func() { c.n -= d })
}

// buffer is a text model whose keystrokes coalesce into a single undo step.
type buffer struct{ s string }

func (b *buffer) typeStr(r string) undo.Command {
	return undo.NewCoalescing("type", "Typing",
		func() { b.s += r },
		func() { b.s = b.s[:len(b.s)-len(r)] })
}

func TestPush_AppliesAndRecords(t *testing.T) {
	c := &counter{}
	s := undo.New()
	if s.CanUndo() || s.CanRedo() {
		t.Fatal("fresh stack must have nothing to undo/redo")
	}
	s.Push(c.add(3))
	if c.n != 3 {
		t.Fatalf("Push must apply Do: n = %d, want 3", c.n)
	}
	if !s.CanUndo() || s.CanRedo() {
		t.Fatal("after one push: CanUndo true, CanRedo false")
	}
	if s.Len() != 1 || s.Cursor() != 1 {
		t.Fatalf("Len/Cursor = %d/%d, want 1/1", s.Len(), s.Cursor())
	}
}

func TestUndoRedo_RestoresExactState(t *testing.T) {
	c := &counter{}
	s := undo.New()
	s.Push(c.add(10)) // n = 10
	s.Push(c.add(3))  // n = 13
	s.Push(c.add(-5)) // n = 8
	if c.n != 8 {
		t.Fatalf("n = %d, want 8", c.n)
	}

	if !s.Undo() || c.n != 13 {
		t.Fatalf("undo #1: n = %d, want 13", c.n)
	}
	if !s.Undo() || c.n != 10 {
		t.Fatalf("undo #2: n = %d, want 10", c.n)
	}
	if !s.Undo() || c.n != 0 {
		t.Fatalf("undo #3: n = %d, want 0", c.n)
	}
	if s.Undo() {
		t.Fatal("undo past the start must return false")
	}
	if c.n != 0 {
		t.Fatalf("n after over-undo = %d, want 0", c.n)
	}

	if !s.Redo() || c.n != 10 {
		t.Fatalf("redo #1: n = %d, want 10", c.n)
	}
	if !s.Redo() || c.n != 13 {
		t.Fatalf("redo #2: n = %d, want 13", c.n)
	}
	if !s.Redo() || c.n != 8 {
		t.Fatalf("redo #3: n = %d, want 8", c.n)
	}
	if s.Redo() {
		t.Fatal("redo past the end must return false")
	}
	if c.n != 8 {
		t.Fatalf("n after over-redo = %d, want 8", c.n)
	}
}

func TestPush_DiscardsRedoTail(t *testing.T) {
	c := &counter{}
	s := undo.New()
	s.Push(c.add(1)) // n = 1
	s.Push(c.add(2)) // n = 3
	s.Push(c.add(4)) // n = 7
	s.Undo()         // n = 3
	s.Undo()         // n = 1
	if !s.CanRedo() {
		t.Fatal("expected a redo tail before the divergent push")
	}
	s.Push(c.add(100)) // diverge; n = 101
	if c.n != 101 {
		t.Fatalf("n = %d, want 101", c.n)
	}
	if s.CanRedo() {
		t.Fatal("divergent push must discard the redo tail")
	}
	if s.Len() != 2 || s.Cursor() != 2 {
		t.Fatalf("Len/Cursor = %d/%d, want 2/2", s.Len(), s.Cursor())
	}
	// The discarded add(2)/add(4) can no longer be redone; undo unwinds only the
	// surviving add(1) and add(100).
	s.Undo() // n = 1
	s.Undo() // n = 0
	if c.n != 0 {
		t.Fatalf("n = %d, want 0", c.n)
	}
}

func TestCoalescing_FoldsRunIntoOneStep(t *testing.T) {
	b := &buffer{}
	s := undo.New() // coalescing on by default
	for _, r := range []string{"h", "e", "l", "l", "o"} {
		s.Push(b.typeStr(r))
	}
	if b.s != "hello" {
		t.Fatalf("s = %q, want %q", b.s, "hello")
	}
	if s.Len() != 1 {
		t.Fatalf("coalesced run must be one step, Len = %d", s.Len())
	}
	if s.UndoLabel() != "Typing" {
		t.Fatalf("UndoLabel = %q, want %q", s.UndoLabel(), "Typing")
	}
	if !s.Undo() || b.s != "" {
		t.Fatalf("single undo must clear the whole run: s = %q", b.s)
	}
	if !s.Redo() || b.s != "hello" {
		t.Fatalf("single redo must restore the whole run: s = %q", b.s)
	}
}

func TestCoalescing_Disabled(t *testing.T) {
	b := &buffer{}
	s := undo.New(undo.WithCoalescing(false))
	for _, r := range []string{"h", "i"} {
		s.Push(b.typeStr(r))
	}
	if s.Len() != 2 {
		t.Fatalf("coalescing off must keep steps separate, Len = %d", s.Len())
	}
	s.Undo()
	if b.s != "h" {
		t.Fatalf("one undo removes one char: s = %q, want %q", b.s, "h")
	}
	s.Undo()
	if b.s != "" {
		t.Fatalf("second undo empties: s = %q", b.s)
	}
}

func TestCoalescing_DifferentKindsDoNotFold(t *testing.T) {
	var log string
	typeCmd := undo.NewCoalescing("type", "Typing",
		func() { log += "t" }, func() { log = log[:len(log)-1] })
	delCmd := undo.NewCoalescing("delete", "Delete",
		func() { log += "d" }, func() { log = log[:len(log)-1] })
	s := undo.New()
	s.Push(typeCmd)
	s.Push(delCmd) // different key ⇒ new step
	if s.Len() != 2 {
		t.Fatalf("different kinds must not fold, Len = %d", s.Len())
	}
}

func TestPush_NonCoalescerTopAppends(t *testing.T) {
	n := 0
	s := undo.New()
	s.Push(plainCmd{n: &n, label: "one"}) // plainCmd is not a Coalescer
	s.Push(plainCmd{n: &n, label: "two"})
	if s.Len() != 2 {
		t.Fatalf("non-coalescable commands must stay separate, Len = %d", s.Len())
	}
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
}

func TestPush_CoalescerTopButForeignNextAppends(t *testing.T) {
	n := 0
	s := undo.New()
	// Top IS a Coalescer (funcCommand) but next is a foreign type → merge fails,
	// so the stack appends.
	s.Push(undo.NewCoalescing("type", "Typing", func() { n++ }, func() { n-- }))
	s.Push(plainCmd{n: &n, label: "foreign"})
	if s.Len() != 2 {
		t.Fatalf("foreign next must append, Len = %d", s.Len())
	}
}

func TestWithLimit_DropsOldest(t *testing.T) {
	c := &counter{}
	s := undo.New(undo.WithLimit(2))
	s.Push(c.add(1)) // n = 1
	s.Push(c.add(2)) // n = 3
	s.Push(c.add(4)) // n = 7 ; add(1) dropped
	if s.Len() != 2 {
		t.Fatalf("limit must cap retained commands, Len = %d", s.Len())
	}
	if c.n != 7 {
		t.Fatalf("dropping history must not change state: n = %d, want 7", c.n)
	}
	if !s.Undo() || c.n != 3 {
		t.Fatalf("undo #1: n = %d, want 3", c.n)
	}
	if !s.Undo() || c.n != 1 {
		t.Fatalf("undo #2: n = %d, want 1", c.n)
	}
	if s.Undo() {
		t.Fatal("the dropped oldest command must not be undoable")
	}
	if c.n != 1 {
		t.Fatalf("n after exhausting undo = %d, want 1", c.n)
	}
}

func TestWithLimit_UnderCapKeepsAll(t *testing.T) {
	c := &counter{}
	s := undo.New(undo.WithLimit(5))
	s.Push(c.add(1))
	s.Push(c.add(1))
	if s.Len() != 2 {
		t.Fatalf("under the cap nothing drops, Len = %d", s.Len())
	}
	// A non-positive limit means unlimited.
	s2 := undo.New(undo.WithLimit(0))
	for i := 0; i < 20; i++ {
		s2.Push(c.add(1))
	}
	if s2.Len() != 20 {
		t.Fatalf("limit<=0 must be unlimited, Len = %d", s2.Len())
	}
}

func TestClear_EmptiesHistory(t *testing.T) {
	c := &counter{}
	s := undo.New()
	s.Push(c.add(5)) // n = 5
	s.Clear()
	if s.Len() != 0 || s.CanUndo() || s.CanRedo() {
		t.Fatal("Clear must empty the history")
	}
	if c.n != 5 {
		t.Fatalf("Clear must not touch state: n = %d, want 5", c.n)
	}
}

func TestLabels_EmptyAtEnds(t *testing.T) {
	c := &counter{}
	s := undo.New()
	if s.UndoLabel() != "" || s.RedoLabel() != "" {
		t.Fatal("empty stack has no labels")
	}
	s.Push(c.add(1))
	if s.UndoLabel() != "Add 1" {
		t.Fatalf("UndoLabel = %q, want %q", s.UndoLabel(), "Add 1")
	}
	if s.RedoLabel() != "" {
		t.Fatalf("RedoLabel = %q, want empty", s.RedoLabel())
	}
	s.Undo()
	if s.UndoLabel() != "" {
		t.Fatalf("UndoLabel = %q, want empty", s.UndoLabel())
	}
	if s.RedoLabel() != "Add 1" {
		t.Fatalf("RedoLabel = %q, want %q", s.RedoLabel(), "Add 1")
	}
}

func TestMVVMCommands_TrackAvailability(t *testing.T) {
	c := &counter{}
	s := undo.New()
	uc, rc := s.UndoCommand(), s.RedoCommand()

	if uc.CanExecute() || rc.CanExecute() {
		t.Fatal("nothing to undo/redo yet")
	}
	s.Push(c.add(7)) // n = 7
	if !uc.CanExecute() || rc.CanExecute() {
		t.Fatal("after push: undo enabled, redo disabled")
	}
	uc.Execute() // undo through the bound command → n = 0
	if c.n != 0 {
		t.Fatalf("undo command must revert: n = %d, want 0", c.n)
	}
	if uc.CanExecute() || !rc.CanExecute() {
		t.Fatal("after undo: undo disabled, redo enabled")
	}
	rc.Execute() // redo through the bound command → n = 7
	if c.n != 7 {
		t.Fatalf("redo command must re-apply: n = %d, want 7", c.n)
	}
}

func TestObservableBindings_ReflectState(t *testing.T) {
	c := &counter{}
	s := undo.New()

	var canUndo bool
	var undoText string
	s.CanUndoBinding().Subscribe(func(v bool) { canUndo = v })
	s.UndoTextBinding().Subscribe(func(v string) { undoText = v })

	// Initial published values.
	if s.CanUndoBinding().Get() {
		t.Fatal("initial CanUndo must be false")
	}
	if s.CanRedoBinding().Get() {
		t.Fatal("initial CanRedo must be false")
	}
	if s.UndoTextBinding().Get() != "Undo" || s.RedoTextBinding().Get() != "Redo" {
		t.Fatalf("initial captions = %q/%q, want Undo/Redo",
			s.UndoTextBinding().Get(), s.RedoTextBinding().Get())
	}

	s.Push(c.add(1))
	if !canUndo {
		t.Fatal("CanUndo observable must fire true on push")
	}
	if undoText != "Undo Add 1" {
		t.Fatalf("UndoText observable = %q, want %q", undoText, "Undo Add 1")
	}
	if s.RedoTextBinding().Get() != "Redo" {
		t.Fatalf("RedoText = %q, want %q", s.RedoTextBinding().Get(), "Redo")
	}

	s.Undo()
	if canUndo {
		t.Fatal("CanUndo observable must fire false after undo-to-empty")
	}
	if undoText != "Undo" {
		t.Fatalf("UndoText after undo = %q, want %q", undoText, "Undo")
	}
	if s.RedoTextBinding().Get() != "Redo Add 1" {
		t.Fatalf("RedoText = %q, want %q", s.RedoTextBinding().Get(), "Redo Add 1")
	}
}
