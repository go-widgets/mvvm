// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package undo_test

import (
	"testing"

	"github.com/go-widgets/mvvm/undo"
)

// plainCmd is a Command that does NOT implement Coalescer — used to exercise the
// stack's "top is not coalescable" and "next is a foreign type" branches.
type plainCmd struct {
	n     *int
	label string
}

func (p plainCmd) Do()           { *p.n++ }
func (p plainCmd) Undo()         { *p.n-- }
func (p plainCmd) Label() string { return p.label }

func TestNew_DoUndoLabel(t *testing.T) {
	n := 0
	c := undo.NewCommand("inc", func() { n += 5 }, func() { n -= 5 })
	if got := c.Label(); got != "inc" {
		t.Fatalf("Label = %q, want %q", got, "inc")
	}
	c.Do()
	if n != 5 {
		t.Fatalf("after Do, n = %d, want 5", n)
	}
	c.Undo()
	if n != 0 {
		t.Fatalf("after Undo, n = %d, want 0", n)
	}
}

func TestCoalesceWith_ForeignTypeDoesNotMerge(t *testing.T) {
	n := 0
	top := undo.NewCoalescing("k", "typing", func() { n++ }, func() { n-- })
	c, ok := top.(undo.Coalescer)
	if !ok {
		t.Fatal("NewCoalescing command must implement Coalescer")
	}
	if _, merged := c.CoalesceWith(plainCmd{n: &n}); merged {
		t.Fatal("coalescing with a foreign command type must fail")
	}
}

func TestCoalesceWith_EmptyKeyDoesNotMerge(t *testing.T) {
	n := 0
	top := undo.NewCoalescing("", "typing", func() { n++ }, func() { n-- }).(undo.Coalescer)
	next := undo.NewCoalescing("", "typing", func() { n++ }, func() { n-- })
	if _, ok := top.CoalesceWith(next); ok {
		t.Fatal("empty key must never coalesce")
	}
}

func TestCoalesceWith_DifferentKeyDoesNotMerge(t *testing.T) {
	n := 0
	top := undo.NewCoalescing("a", "x", func() { n++ }, func() { n-- }).(undo.Coalescer)
	next := undo.NewCoalescing("b", "y", func() { n++ }, func() { n-- })
	if _, ok := top.CoalesceWith(next); ok {
		t.Fatal("different keys must not coalesce")
	}
}

func TestCoalesceWith_MergesChainedEffects(t *testing.T) {
	var log []string
	mk := func(id string) undo.Command {
		return undo.NewCoalescing("k", id,
			func() { log = append(log, "do:"+id) },
			func() { log = append(log, "undo:"+id) })
	}
	top := mk("1").(undo.Coalescer)
	merged, ok := top.CoalesceWith(mk("2"))
	if !ok {
		t.Fatal("same-key commands must coalesce")
	}
	if merged.Label() != "2" {
		t.Fatalf("merged Label = %q, want %q (newest)", merged.Label(), "2")
	}
	log = nil
	merged.Do() // redo applies oldest-first
	if len(log) != 2 || log[0] != "do:1" || log[1] != "do:2" {
		t.Fatalf("merged Do order = %v, want [do:1 do:2]", log)
	}
	log = nil
	merged.Undo() // undo reverts newest-first
	if len(log) != 2 || log[0] != "undo:2" || log[1] != "undo:1" {
		t.Fatalf("merged Undo order = %v, want [undo:2 undo:1]", log)
	}
}
