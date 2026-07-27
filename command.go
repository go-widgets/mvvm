// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

// Command is an invocable action with an optional CanExecute predicate — the
// "command" primitive of MVVM. Bind it to a button-shaped widget with
// BindCommand: the widget invokes Execute, and a change in CanExecute re-greys
// or re-enables the widget.
type Command struct {
	exec    func()
	canExec func() bool // nil ⇒ always executable
	ceSubs  map[int]func()
	nextID  int
}

// NewCommand builds a Command from an action and an optional CanExecute
// predicate (pass nil to mean "always executable").
func NewCommand(exec func(), canExec func() bool) *Command {
	return &Command{exec: exec, canExec: canExec}
}

// CanExecute reports whether the command may run right now.
func (c *Command) CanExecute() bool { return c.canExec == nil || c.canExec() }

// Execute runs the action, but only when CanExecute is true — so it is safe to
// wire straight to a widget's OnClick without a guard at the call site.
func (c *Command) Execute() {
	if c.exec != nil && c.CanExecute() {
		c.exec()
	}
}

// RaiseCanExecuteChanged notifies every binding to re-query CanExecute. Call it
// when whatever the predicate reads has changed; BindCanExecute wires this up
// automatically for Observable sources.
func (c *Command) RaiseCanExecuteChanged() {
	for _, fn := range c.ceSubs {
		fn()
	}
}

// SubscribeCanExecuteChanged registers fn, called on each
// RaiseCanExecuteChanged, and returns an unsubscribe.
func (c *Command) SubscribeCanExecuteChanged(fn func()) (unsubscribe func()) {
	if c.ceSubs == nil {
		c.ceSubs = map[int]func(){}
	}
	id := c.nextID
	c.nextID++
	c.ceSubs[id] = fn
	return func() { delete(c.ceSubs, id) }
}

// BindCanExecute makes the command re-raise CanExecuteChanged whenever any of
// the given sources changes, so a bound button re-greys automatically as the
// ViewModel state its predicate depends on evolves. Returns an unbind that
// detaches from every source.
func BindCanExecute(c *Command, sources ...Changeable) (unbind func()) {
	unsubs := make([]func(), 0, len(sources))
	for _, s := range sources {
		unsubs = append(unsubs, s.SubscribeChanged(c.RaiseCanExecuteChanged))
	}
	return func() {
		for _, u := range unsubs {
			u()
		}
	}
}
