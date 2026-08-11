// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package undo_test

import (
	"fmt"

	"github.com/go-widgets/mvvm/undo"
)

// ExampleStack shows an app recording edits, then undoing and redoing them. The
// document below stands in for any model; each command captures the exact
// inverse so state round-trips.
func ExampleStack() {
	doc := "" // the model
	s := undo.New()

	write := func(text string) undo.Command {
		return undo.NewCommand("Write "+text,
			func() { doc += text },
			func() { doc = doc[:len(doc)-len(text)] })
	}

	s.Push(write("hello"))
	s.Push(write(" world"))
	fmt.Printf("%q — %s\n", doc, s.UndoTextBinding().Get())

	s.Undo()
	fmt.Printf("%q — %s\n", doc, s.UndoTextBinding().Get())

	s.Redo()
	fmt.Printf("%q\n", doc)
	// Output:
	// "hello world" — Undo Write  world
	// "hello" — Undo Write hello
	// "hello world"
}

// ExampleNewCoalescing shows a run of keystrokes collapsing into one undo step.
func ExampleNewCoalescing() {
	doc := ""
	s := undo.New()

	key := func(r string) undo.Command {
		return undo.NewCoalescing("type", "Typing",
			func() { doc += r },
			func() { doc = doc[:len(doc)-len(r)] })
	}

	for _, r := range []string{"h", "i", "!"} {
		s.Push(key(r))
	}
	fmt.Printf("%q steps=%d\n", doc, s.Len())

	s.Undo() // one undo reverts the whole run
	fmt.Printf("%q\n", doc)
	// Output:
	// "hi!" steps=1
	// ""
}
