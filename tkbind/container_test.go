// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// probe is a minimal toolkit.Widget (Base supplies every method) used to observe
// how a Container lays its children out.
type probe struct{ toolkit.Base }

// TestBindContainer checks the container's items follow the observable list —
// seeded, rebuilt on change, and detached on unbind — and covers the invalidate
// and nil-invalidate paths.
func TestBindContainer(t *testing.T) {
	l := mvvm.NewObservableList[string]("a", "b")
	c := toolkit.NewContainer(toolkit.FitLayout{})
	c.SetBounds(toolkit.Rect{X: 0, Y: 0, W: 10, H: 10})

	repaints := 0
	factory := func(string) toolkit.Item { return toolkit.Item{Widget: &probe{}} }
	unbind := BindContainer(l, c, factory, func() { repaints++ })

	if len(c.Items()) != 2 {
		t.Fatalf("seed: %d items, want 2", len(c.Items()))
	}
	if repaints != 1 {
		t.Fatalf("seed should invalidate once, got %d", repaints)
	}

	l.Append("c")
	if len(c.Items()) != 3 || repaints != 2 {
		t.Fatalf("after append: items=%d repaints=%d, want 3/2", len(c.Items()), repaints)
	}

	unbind()
	l.Append("d")
	if len(c.Items()) != 3 {
		t.Fatalf("after unbind the container must not rebuild: %d", len(c.Items()))
	}

	// nil invalidate must not panic.
	c2 := toolkit.NewContainer(toolkit.FitLayout{})
	u2 := BindContainer(mvvm.NewObservableList[string]("x"), c2, factory, nil)
	if len(c2.Items()) != 1 {
		t.Fatalf("nil-invalidate seed: %d", len(c2.Items()))
	}
	u2()
}

// TestBindCardActive checks the CardLayout's Active follows the observable and the
// active card is the one laid out, with unbind detaching and nil-invalidate safe.
func TestBindCardActive(t *testing.T) {
	sel := mvvm.NewObservable(0)
	a, b := &probe{}, &probe{}
	card := &toolkit.CardLayout{}
	c := toolkit.NewContainer(card)
	c.AddWidget(a).AddWidget(b)
	c.SetBounds(toolkit.Rect{X: 0, Y: 0, W: 10, H: 10})

	repaints := 0
	unbind := BindCardActive(sel, c, card, func() { repaints++ })

	// Seed: card 0 active → a filled, b empty.
	if card.Active != 0 || a.Bounds().W == 0 || b.Bounds() != (toolkit.Rect{}) {
		t.Fatalf("seed: active=%d a=%+v b=%+v", card.Active, a.Bounds(), b.Bounds())
	}
	if repaints != 1 {
		t.Fatalf("seed should invalidate once, got %d", repaints)
	}

	sel.Set(1) // switch to card 1 → b filled, a empty
	if card.Active != 1 || b.Bounds().W == 0 || a.Bounds() != (toolkit.Rect{}) {
		t.Fatalf("switch: active=%d a=%+v b=%+v", card.Active, a.Bounds(), b.Bounds())
	}
	if repaints != 2 {
		t.Fatalf("switch should invalidate, got %d", repaints)
	}

	unbind()
	sel.Set(0)
	if card.Active != 1 {
		t.Fatalf("after unbind Active must not change: %d", card.Active)
	}

	// nil invalidate must not panic.
	BindCardActive(mvvm.NewObservable(0), toolkit.NewContainer(&toolkit.CardLayout{}), &toolkit.CardLayout{}, nil)
}
