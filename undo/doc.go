// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package undo is a render-agnostic undo/redo command stack for the go-widgets
// MVVM layer. It has no dependency on any backend: it builds only on the
// dependency-free primitives of the parent [github.com/go-widgets/mvvm] package
// (Observable and Command), so the same stack drives pixel (toolkit) and
// terminal-cell (tui) views alike.
//
// # Model
//
// A [Command] is a reversible action: Do applies it (and re-applies it on
// redo), Undo reverts it, and Label names it for the UI. A [Stack] records the
// commands the user performs and moves a cursor across them: [Stack.Undo] steps
// the cursor back and reverts, [Stack.Redo] steps it forward and re-applies.
// Pushing a fresh command after some undos discards the redo tail, exactly like
// every editor.
//
// # Coalescing
//
// Contiguous commands of the same kind can collapse into a single undo step —
// so a run of keystrokes or a drag gesture undoes in one go rather than one
// character or pixel at a time. A command opts in by implementing [Coalescer];
// [NewCoalescing] provides a ready-made key-matched implementation.
//
// # Binding to the UI
//
// A [Stack] exposes its state as MVVM primitives so an "Undo"/"Redo" button or
// menu item binds with no glue: [Stack.UndoCommand]/[Stack.RedoCommand] are
// [mvvm.Command]s (their CanExecute tracks CanUndo/CanRedo), and
// [Stack.CanUndoBinding], [Stack.UndoTextBinding], and their redo twins are
// [mvvm.Observable]s carrying the enabled flag and the live "Undo <label>"
// caption. Wire them with the parent package's BindCommand / OneWay adapters.
//
// # Threading
//
// Like the rest of the MVVM layer, a Stack is not safe for concurrent use —
// drive it from the UI goroutine.
package undo
