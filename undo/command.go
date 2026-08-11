// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package undo

// Command is a single reversible action recorded on a [Stack].
//
// Do applies the action; it is called once when the command is pushed and again
// on every redo. Undo reverts it, restoring the exact state that existed before
// Do ran. Label names the action for the UI (e.g. "Typing", "Delete row"); it
// may be empty. Do and Undo must be exact inverses so that any Do/Undo or
// Undo/Redo pair round-trips.
type Command interface {
	Do()
	Undo()
	Label() string
}

// Coalescer is the optional interface a [Command] implements to merge with the
// command immediately preceding it on the stack. When [Stack.Push] is about to
// record next and the current top command implements Coalescer, it calls
// top.CoalesceWith(next): returning (merged, true) replaces the top in place so
// the two collapse into a single undo step, while (nil, false) records next as
// a new step. This is how a run of same-kind commands (keystrokes, drag deltas)
// undoes in one action.
type Coalescer interface {
	CoalesceWith(next Command) (merged Command, ok bool)
}

// funcCommand is a Command built from Do/Undo closures. A non-empty key opts it
// into coalescing with a preceding funcCommand carrying the same key.
type funcCommand struct {
	key   string
	label string
	do    func()
	undo  func()
}

// NewCommand returns a non-coalescing [Command] that runs do on Do (and redo),
// runs undo on Undo, and reports label. do and undo must be non-nil and exact
// inverses.
func NewCommand(label string, do, undo func()) Command {
	return &funcCommand{label: label, do: do, undo: undo}
}

// NewCoalescing returns a [Command] like [NewCommand] that also coalesces: when it is
// pushed immediately after another NewCoalescing command sharing the same
// non-empty key, the two collapse into one undo step whose Undo reverts both
// newest-first and whose Do re-applies both oldest-first. An empty key never
// coalesces (behaving like [New]). Coalescing re-applies on each further push,
// so an entire run folds into a single step.
func NewCoalescing(key, label string, do, undo func()) Command {
	return &funcCommand{key: key, label: label, do: do, undo: undo}
}

func (c *funcCommand) Do()           { c.do() }
func (c *funcCommand) Undo()         { c.undo() }
func (c *funcCommand) Label() string { return c.label }

// CoalesceWith merges next into c when both are funcCommands sharing the same
// non-empty key. The merged command chains the two effects: Do runs c.do then
// next's do; Undo runs next's undo then c.undo. It carries next's key and label
// so a further same-key push folds in too.
func (c *funcCommand) CoalesceWith(next Command) (Command, bool) {
	n, ok := next.(*funcCommand)
	if !ok || c.key == "" || c.key != n.key {
		return nil, false
	}
	cdo, ndo := c.do, n.do
	cundo, nundo := c.undo, n.undo
	return &funcCommand{
		key:   n.key,
		label: n.label,
		do:    func() { cdo(); ndo() },
		undo:  func() { nundo(); cundo() },
	}, true
}
