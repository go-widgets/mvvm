// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

func TestBindCycleClampsAndGoesBothWays(t *testing.T) {
	at := mvvm.NewObservable(9)
	c := toolkit.NewCycleButton("one", "two", "three")
	unbind := BindCycle(at, c, nil)

	if got := c.Index().Get(); got != 2 {
		t.Errorf("the button is at %d for a model asking for position 9 of three", got)
	}
	if got := at.Get(); got != 2 {
		t.Errorf("the model still holds %d, which is not a position of the three", got)
	}
	c.Index().Set(1)
	if at.Get() != 1 {
		t.Errorf("the button was cycled to 1 and the model holds %d", at.Get())
	}
	at.Set(0)
	if c.Index().Get() != 0 {
		t.Errorf("the model chose 0 and the button is at %d", c.Index().Get())
	}
	unbind()
	at.Set(2)
	if c.Index().Get() != 0 {
		t.Error("the model still reaches the button after unbind")
	}
}

func TestBindCycleValuesMapsPositionsToWhatTheyMean(t *testing.T) {
	// The case this was written for: a quarter turn. The button sits at 0, 1
	// or 2 and the document wants 90, 180 or 270, and an app with no binder
	// for it writes `90 * (i + 1)` in one direction and nothing in the other.
	turn := mvvm.NewObservable(180)
	c := toolkit.NewCycleButton("a quarter", "a half", "three quarters")
	repaints := 0
	unbind := BindCycleValues(turn, c, []int{90, 180, 270}, func() { repaints++ })

	if got := c.Index().Get(); got != 1 {
		t.Errorf("a model holding 180 put the button at %d", got)
	}
	if got := c.Value(); got != "a half" {
		t.Errorf("the button reads %q", got)
	}

	c.Index().Set(2)
	if got := turn.Get(); got != 270 {
		t.Errorf("the button was cycled to the third position and the model holds %d", got)
	}
	turn.Set(90)
	if got := c.Index().Get(); got != 0 {
		t.Errorf("the model was set to 90 and the button is at %d", got)
	}
	if repaints == 0 {
		t.Error("nothing asked for a repaint, so a bound button would never be redrawn")
	}

	unbind()
	turn.Set(270)
	if c.Index().Get() != 0 {
		t.Error("the model still reaches the button after unbind")
	}
	c.Index().Set(1)
	if turn.Get() != 270 {
		t.Error("the button still reaches the model after unbind")
	}
}

func TestBindCycleValuesSettlesAValueTheTableHasNot(t *testing.T) {
	// ⛔ A value with no position cannot be shown, and leaving it in the model
	// is how two sides come to hold different answers: the button would sit at
	// the first position and the model would still say 45, so the next thing
	// to read the model turns the page by an angle nobody chose.
	turn := mvvm.NewObservable(45)
	c := toolkit.NewCycleButton("a quarter", "a half", "three quarters")
	BindCycleValues(turn, c, []int{90, 180, 270}, nil)()

	if got := c.Index().Get(); got != 0 {
		t.Errorf("the button is at %d for a value the table has not got", got)
	}
	if got := turn.Get(); got != 90 {
		t.Errorf("the model still holds %d, which no position of this button means", got)
	}
}

func TestBindCycleValuesWithNoTableBindsNothing(t *testing.T) {
	// A table with no entries cannot say what any position means, so it binds
	// nothing rather than settling the model on an element that is not there.
	v := mvvm.NewObservable("kept")
	c := toolkit.NewCycleButton("one", "two")
	BindCycleValues(v, c, nil, nil)()
	if v.Get() != "kept" {
		t.Errorf("an empty table moved the model to %q", v.Get())
	}
	c.Index().Set(1)
	if v.Get() != "kept" {
		t.Errorf("an empty table let the button write %q into the model", v.Get())
	}
}

func TestBindButtonExecutesAndGreysItself(t *testing.T) {
	ran := 0
	allowed := true
	cmd := mvvm.NewCommand(func() { ran++ }, func() bool { return allowed })
	b := toolkit.NewButton("Delete", nil)
	repaints := 0
	unbind := BindButton(cmd, b, func() { repaints++ })

	if b.Disabled().Get() {
		t.Error("the button starts disabled though the command says it can run")
	}
	b.OnClick()
	if ran != 1 {
		t.Errorf("pressing it ran the command %d times", ran)
	}

	// The rule changes, and the button follows without anybody telling it.
	allowed = false
	cmd.RaiseCanExecuteChanged()
	if !b.Disabled().Get() {
		t.Error("the command can no longer run and the button is still pressable")
	}
	if repaints == 0 {
		t.Error("nothing asked for a repaint, so the button would stay looking pressable")
	}
	b.OnClick()
	if ran != 1 {
		t.Errorf("a command that cannot run ran anyway: %d times", ran)
	}

	unbind()
	allowed = true
	cmd.RaiseCanExecuteChanged()
	if !b.Disabled().Get() {
		t.Error("the command still reaches the button after unbind")
	}
}

func TestTheCycleAndCommandBindersSurviveANilInvalidate(t *testing.T) {
	BindCycle(mvvm.NewObservable(0), toolkit.NewCycleButton("a", "b"), nil)()
	BindCycleValues(mvvm.NewObservable(1), toolkit.NewCycleButton("a", "b"), []int{1, 2}, nil)()
	BindButton(mvvm.NewCommand(func() {}, nil), toolkit.NewButton("x", nil), nil)()
}

func TestBindCycleClampsAtBothEnds(t *testing.T) {
	// Both ends: the `if` arm and the `else if` arm are different code, and a
	// suite that only ever asks for a position past the last never runs the
	// one that catches a position before the first.
	below := mvvm.NewObservable(-4)
	c := toolkit.NewCycleButton("one", "two", "three")
	BindCycle(below, c, nil)()
	if below.Get() != 0 || c.Index().Get() != 0 {
		t.Errorf("a model asking for position -4 left the model at %d and the button at %d",
			below.Get(), c.Index().Get())
	}

	// A button with no options has nothing to clamp to, so the index is left
	// alone rather than reached for.
	none := mvvm.NewObservable(5)
	BindCycle(none, toolkit.NewCycleButton(), nil)()
	if none.Get() != 5 {
		t.Errorf("a button with no options moved the index to %d", none.Get())
	}
}

func TestAPositionWithNoValueDoesNotTakeTheProgramWithIt(t *testing.T) {
	// ⛔ Index() is an Observable and Observables have no guard: anything
	// holding the button can Set a position the table has no value for, and
	// values[i] on that position is a panic, not a wrong answer. The button
	// itself only ever cycles within its options, so this costs nothing in
	// normal use and is the whole difference between a mistake and a crash.
	//
	// It survived the first mutation run -- `if true` in place of the range
	// check broke nothing -- because every other test here drives the button
	// the way the button drives itself.
	turn := mvvm.NewObservable(90)
	c := toolkit.NewCycleButton("a quarter", "a half")
	defer BindCycleValues(turn, c, []int{90, 180}, nil)()

	c.Index().Set(7)
	if got := turn.Get(); got != 90 {
		t.Errorf("a position the table has no value for put %d into the model", got)
	}
	c.Index().Set(-1)
	if got := turn.Get(); got != 90 {
		t.Errorf("a negative position put %d into the model", got)
	}
}
