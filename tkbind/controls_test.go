// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

func TestBindEntryGoesBothWaysAndStops(t *testing.T) {
	text := mvvm.NewObservable("from the model")
	e := toolkit.NewEntry("from the widget")
	repaints := 0
	unbind := BindEntry(text, e, func() { repaints++ })

	if got := e.Text().Get(); got != "from the model" {
		t.Errorf("the box shows %q, and the ViewModel is what it is bound to", got)
	}
	text.Set("the model changed")
	if got := e.Text().Get(); got != "the model changed" {
		t.Errorf("the model changed and the box shows %q", got)
	}
	e.Text().Set("somebody typed")
	if got := text.Get(); got != "somebody typed" {
		t.Errorf("somebody typed and the model holds %q", got)
	}
	if repaints == 0 {
		t.Error("nothing asked for a repaint, so a bound control would never be redrawn")
	}

	unbind()
	was := e.Text().Get()
	text.Set("after the unbind")
	if e.Text().Get() != was {
		t.Error("the model still reaches the box after unbind")
	}
	e.Text().Set("typed after the unbind")
	if text.Get() != "after the unbind" {
		t.Error("the box still reaches the model after unbind")
	}
}

func TestBindSpinClampsTheModelThroughTheWidget(t *testing.T) {
	// ⛔ The seed is the point. Value().Set does NOT clamp -- only SetValue and
	// the +/- and key paths do -- so a ViewModel holding a number outside the
	// control's range would leave the two sides disagreeing about what is on
	// the screen, with the ViewModel's illegal value still in it.
	count := mvvm.NewObservable(500)
	sp := toolkit.NewSpinButton(1, 10, 3, 1)
	unbind := BindSpin(count, sp, nil)

	if got := sp.Value().Get(); got != 10 {
		t.Errorf("the control shows %d for a model asking 500 of a maximum 10", got)
	}
	if got := count.Get(); got != 10 {
		t.Errorf("the model still holds %d, which is not a value the control can show", got)
	}

	sp.Value().Set(4)
	if got := count.Get(); got != 4 {
		t.Errorf("the control was stepped to 4 and the model holds %d", got)
	}
	count.Set(7)
	if got := sp.Value().Get(); got != 7 {
		t.Errorf("the model was set to 7 and the control shows %d", got)
	}

	unbind()
	count.Set(2)
	if sp.Value().Get() != 7 {
		t.Error("the model still reaches the control after unbind")
	}
}

func TestBindChoiceSettlesOnWhatTheListCanActuallyOffer(t *testing.T) {
	// ⛔ Select REFUSES an index outside the options rather than clamping it,
	// so seeding through it is not enough on its own: without pushing back
	// what the widget settled on, a ViewModel asking for option nine of three
	// leaves the list where it was and the ViewModel still holding nine --
	// two sides, two answers, and the next thing that reads the model is wrong.
	pick := mvvm.NewObservable(9)
	d := toolkit.NewDropDown([]string{"first", "second", "third"}, 1)
	unbind := BindChoice(pick, d, nil)

	if got := d.Selected().Get(); got != 2 {
		t.Errorf("the list is on %d for a model asking for an option it has not got", got)
	}
	if got := pick.Get(); got != 2 {
		t.Errorf("the model still holds %d, which is not an option of the three", got)
	}
	if got := d.Current(); got != "third" {
		t.Errorf("the list reads %q", got)
	}

	d.Select(1)
	if got := pick.Get(); got != 1 {
		t.Errorf("the second option was chosen and the model holds %d", got)
	}
	pick.Set(0)
	if got := d.Selected().Get(); got != 0 {
		t.Errorf("the model chose the first option and the list is on %d", got)
	}

	unbind()
	pick.Set(2)
	if d.Selected().Get() != 0 {
		t.Error("the model still reaches the list after unbind")
	}
}

func TestBindCheckGoesBothWays(t *testing.T) {
	on := mvvm.NewObservable(true)
	c := toolkit.NewCheckButton("keep the links", false)
	unbind := BindCheck(on, c, nil)

	if !c.Checked().Get() {
		t.Error("the box is not ticked for a model that says it is")
	}
	c.Checked().Set(false)
	if on.Get() {
		t.Error("the box was unticked and the model still says it is ticked")
	}
	on.Set(true)
	if !c.Checked().Get() {
		t.Error("the model was set and the box did not follow")
	}

	unbind()
	on.Set(false)
	if !c.Checked().Get() {
		t.Error("the model still reaches the box after unbind")
	}
}

func TestEveryControlBinderSurvivesANilInvalidate(t *testing.T) {
	// A host with nothing to repaint -- a test, a headless ViewModel -- passes
	// nil, and a binder that calls it anyway takes the program with it.
	BindEntry(mvvm.NewObservable(""), toolkit.NewEntry(""), nil)()
	BindSpin(mvvm.NewObservable(1), toolkit.NewSpinButton(0, 5, 0, 1), nil)()
	BindChoice(mvvm.NewObservable(0), toolkit.NewDropDown([]string{"a"}, 0), nil)()
	BindCheck(mvvm.NewObservable(false), toolkit.NewCheckButton("", false), nil)()
}

func TestTheClampsWorkAtBothEnds(t *testing.T) {
	// Both ends, because a clamp tested at one of them is a clamp half
	// written: the `else if` arm and the `if` arm are different code, and a
	// suite that only ever asks for too much never runs the one that catches
	// too little.
	low := mvvm.NewObservable(-40)
	sp := toolkit.NewSpinButton(1, 10, 5, 1)
	BindSpin(low, sp, nil)()
	if low.Get() != 1 || sp.Value().Get() != 1 {
		t.Errorf("a model asking for -40 of a minimum 1 left the model at %d and the control at %d",
			low.Get(), sp.Value().Get())
	}

	before := mvvm.NewObservable(-3)
	d := toolkit.NewDropDown([]string{"first", "second"}, 1)
	BindChoice(before, d, nil)()
	if before.Get() != 0 || d.Selected().Get() != 0 {
		t.Errorf("a model asking for option -3 left the model at %d and the list at %d",
			before.Get(), d.Selected().Get())
	}

	// And a list with no options at all has nothing to clamp to, so the index
	// is left alone rather than reached for.
	none := mvvm.NewObservable(7)
	BindChoice(none, toolkit.NewDropDown(nil, 0), nil)()
	if none.Get() != 7 {
		t.Errorf("an empty list moved the index to %d, though it has no option to move it to", none.Get())
	}
}
