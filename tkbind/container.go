// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// BindContainer drives a toolkit.Container's children from an ObservableList: on
// every list change it rebuilds the container's items via factory (each element →
// a layout Item) and calls invalidate. This is the data-driven half of the
// Sencha-style container/layout model — the view's children follow the view
// model's collection. It mirrors mvvm.BindList's full-rebuild shape (the child
// count is view-scale) while producing widget Items rather than strings. Returns
// an unbind that detaches the list subscription.
func BindContainer[T any](l *mvvm.ObservableList[T], c *toolkit.Container, factory func(T) toolkit.Item, invalidate func()) (unbind func()) {
	rebuild := func() {
		src := l.Slice()
		items := make([]toolkit.Item, len(src))
		for i, v := range src {
			items[i] = factory(v)
		}
		c.SetItems(items...)
		if invalidate != nil {
			invalidate()
		}
	}
	rebuild()
	return l.Subscribe(func(mvvm.ListEvent[T]) { rebuild() })
}

// BindCardActive one-way-binds an Observable[int] to a CardLayout's Active index,
// re-arranging its container so the newly selected card becomes the visible one.
// Which card shows is a view-model concern (like an Ext card layout driven by a
// bound activeItem), so the binding is VM→view only. Returns an unbind that
// detaches the subscription.
func BindCardActive(sel *mvvm.Observable[int], c *toolkit.Container, card *toolkit.CardLayout, invalidate func()) (unbind func()) {
	apply := func(i int) {
		card.Active = i
		c.SetBounds(c.Bounds()) // re-run the layout so the active card is shown
		if invalidate != nil {
			invalidate()
		}
	}
	apply(sel.Get())
	return sel.Subscribe(apply)
}
