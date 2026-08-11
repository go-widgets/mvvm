// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package undo

import "github.com/go-widgets/mvvm"

// Stack is a render-agnostic undo/redo command stack. It holds the sequence of
// commands the user has performed and a cursor marking how many are currently
// applied: everything before the cursor can be undone, everything from the
// cursor on can be redone. Push after an undo discards the redo tail.
//
// A Stack also publishes its state as MVVM primitives (see [Stack.UndoCommand],
// [Stack.CanUndoBinding], [Stack.UndoTextBinding] and their redo twins) so an
// Undo/Redo button or menu binds directly with no extra wiring.
//
// The zero Stack is not usable; construct one with [New]. A Stack is not safe
// for concurrent use.
type Stack struct {
	commands []Command
	cursor   int  // count of applied commands; commands[:cursor] are done
	limit    int  // max retained commands; 0 ⇒ unlimited
	coalesce bool // fold contiguous same-kind commands into one step

	canUndo  *mvvm.Observable[bool]
	canRedo  *mvvm.Observable[bool]
	undoText *mvvm.Observable[string]
	redoText *mvvm.Observable[string]
	undoCmd  *mvvm.Command
	redoCmd  *mvvm.Command
}

// Option configures a [Stack] at construction.
type Option func(*Stack)

// WithLimit caps the number of retained commands at n (n <= 0 ⇒ unlimited, the
// default). When a push would exceed the cap, the oldest commands are dropped —
// they can no longer be undone, but the current state is untouched.
func WithLimit(n int) Option { return func(s *Stack) { s.limit = n } }

// WithCoalescing enables or disables coalescing of contiguous same-kind
// commands (see [Coalescer]). Coalescing is on by default.
func WithCoalescing(enabled bool) Option { return func(s *Stack) { s.coalesce = enabled } }

// New builds an empty [Stack]. By default it is unlimited and coalescing is on;
// pass [WithLimit] / [WithCoalescing] to change either.
func New(opts ...Option) *Stack {
	s := &Stack{
		coalesce: true,
		canUndo:  mvvm.NewObservable(false),
		canRedo:  mvvm.NewObservable(false),
		undoText: mvvm.NewObservable("Undo"),
		redoText: mvvm.NewObservable("Redo"),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.undoCmd = mvvm.NewCommand(func() { s.Undo() }, s.CanUndo)
	s.redoCmd = mvvm.NewCommand(func() { s.Redo() }, s.CanRedo)
	return s
}

// Push applies cmd (calling cmd.Do), discards any redo tail, and records it as
// the newest undoable step. When coalescing is on and the previous top command
// merges with cmd (see [Coalescer]), the two collapse into a single step
// instead of adding one. Honours the configured limit.
func (s *Stack) Push(cmd Command) {
	cmd.Do()
	s.commands = s.commands[:s.cursor] // drop the redo tail
	if s.coalesce && s.cursor > 0 {
		if c, ok := s.commands[s.cursor-1].(Coalescer); ok {
			if merged, ok := c.CoalesceWith(cmd); ok {
				s.commands[s.cursor-1] = merged
				s.refresh()
				return
			}
		}
	}
	s.commands = append(s.commands, cmd)
	s.cursor++
	s.enforceLimit()
	s.refresh()
}

// enforceLimit drops the oldest commands when the limit is exceeded.
func (s *Stack) enforceLimit() {
	if s.limit > 0 && len(s.commands) > s.limit {
		drop := len(s.commands) - s.limit
		s.commands = s.commands[drop:]
		s.cursor -= drop
	}
}

// Undo reverts the most recently applied command and steps the cursor back,
// returning true. It is a no-op returning false when there is nothing to undo.
func (s *Stack) Undo() bool {
	if !s.CanUndo() {
		return false
	}
	s.cursor--
	s.commands[s.cursor].Undo()
	s.refresh()
	return true
}

// Redo re-applies the next command and steps the cursor forward, returning
// true. It is a no-op returning false when there is nothing to redo.
func (s *Stack) Redo() bool {
	if !s.CanRedo() {
		return false
	}
	s.commands[s.cursor].Do()
	s.cursor++
	s.refresh()
	return true
}

// Clear empties the stack, discarding all undo and redo history. The current
// application state is left as-is.
func (s *Stack) Clear() {
	s.commands = nil
	s.cursor = 0
	s.refresh()
}

// CanUndo reports whether there is a command to undo.
func (s *Stack) CanUndo() bool { return s.cursor > 0 }

// CanRedo reports whether there is a command to redo.
func (s *Stack) CanRedo() bool { return s.cursor < len(s.commands) }

// UndoLabel returns the Label of the command that Undo would revert, or "" when
// there is nothing to undo.
func (s *Stack) UndoLabel() string {
	if !s.CanUndo() {
		return ""
	}
	return s.commands[s.cursor-1].Label()
}

// RedoLabel returns the Label of the command that Redo would re-apply, or ""
// when there is nothing to redo.
func (s *Stack) RedoLabel() string {
	if !s.CanRedo() {
		return ""
	}
	return s.commands[s.cursor].Label()
}

// Len reports the number of commands retained (undoable plus redoable).
func (s *Stack) Len() int { return len(s.commands) }

// Cursor reports how many commands are currently applied — the number that can
// be undone.
func (s *Stack) Cursor() int { return s.cursor }

// UndoCommand returns an [mvvm.Command] that undoes one step; its CanExecute
// tracks CanUndo, so a bound button greys out when there is nothing to undo.
func (s *Stack) UndoCommand() *mvvm.Command { return s.undoCmd }

// RedoCommand returns an [mvvm.Command] that redoes one step; its CanExecute
// tracks CanRedo.
func (s *Stack) RedoCommand() *mvvm.Command { return s.redoCmd }

// CanUndoBinding returns an [mvvm.Observable] carrying whether an undo is
// available — bind it to a widget's enabled/visible state.
func (s *Stack) CanUndoBinding() *mvvm.Observable[bool] { return s.canUndo }

// CanRedoBinding returns an [mvvm.Observable] carrying whether a redo is
// available.
func (s *Stack) CanRedoBinding() *mvvm.Observable[bool] { return s.canRedo }

// UndoTextBinding returns an [mvvm.Observable] carrying the live button caption:
// "Undo" when nothing is undoable or the command is unlabelled, else
// "Undo <label>".
func (s *Stack) UndoTextBinding() *mvvm.Observable[string] { return s.undoText }

// RedoTextBinding returns an [mvvm.Observable] carrying the live redo caption,
// "Redo" or "Redo <label>".
func (s *Stack) RedoTextBinding() *mvvm.Observable[string] { return s.redoText }

// refresh republishes the derived observables and re-raises CanExecuteChanged
// on the bound commands after any state change.
func (s *Stack) refresh() {
	s.canUndo.Set(s.CanUndo())
	s.canRedo.Set(s.CanRedo())
	s.undoText.Set(caption("Undo", s.UndoLabel()))
	s.redoText.Set(caption("Redo", s.RedoLabel()))
	s.undoCmd.RaiseCanExecuteChanged()
	s.redoCmd.RaiseCanExecuteChanged()
}

// caption composes a verb and an optional label into a menu/button string.
func caption(verb, label string) string {
	if label == "" {
		return verb
	}
	return verb + " " + label
}
