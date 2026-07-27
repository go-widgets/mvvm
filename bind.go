// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm

// The bind adapters wire an Observable / Command / ObservableList to a widget
// WITHOUT this package importing any widget package. Each adapter reaches the
// widget through a pointer to its value field and a pointer to its callback
// slot — e.g. (&entry.Text, &entry.OnChange). Because go-widgets/toolkit (pixel)
// and go-widgets/tui (cell) mirror the same field names and callback signatures
// per widget, one adapter drives both backends unchanged. The `invalidate` hook
// (nil to skip) requests a repaint after a ViewModel-originated change.

// BindField binds obs two-way to a widget value field and its change-callback
// slot:
//   - seeds the field from the observable (the ViewModel is the source of truth),
//   - composes (does not clobber) any callback already in *hook, so a user edit
//     flows callback → obs.Set,
//   - pushes obs → field on change and calls invalidate.
//
// It is loop-free: the field write is silent (widgets do not notify on direct
// field writes) and Observable.Set skips equal values. The returned unbind
// restores the previous callback and detaches the subscription.
func BindField[T any](obs *Observable[T], field *T, hook *func(T), invalidate func()) (unbind func()) {
	*field = obs.Get()
	prev := *hook
	*hook = func(v T) {
		if prev != nil {
			prev(v)
		}
		obs.Set(v)
	}
	unsub := obs.Subscribe(func(v T) {
		*field = v
		if invalidate != nil {
			invalidate()
		}
	})
	return func() {
		*hook = prev
		unsub()
	}
}

// OneWay binds obs → field only, for view-only sinks that have no user-edit
// callback (a Label's text, a ProgressBar's fraction, a passive Table's
// selection). Returns an unbind that detaches the subscription.
func OneWay[T any](obs *Observable[T], field *T, invalidate func()) (unbind func()) {
	*field = obs.Get()
	return obs.Subscribe(func(v T) {
		*field = v
		if invalidate != nil {
			invalidate()
		}
	})
}

// BindCommand wires a Command to a button-shaped widget: it composes Execute
// into *onClick, and — when setEnabled is non-nil — calls it now and on every
// CanExecuteChanged so the widget reflects executability (a real disable, or a
// backend-specific greying such as swapping a Button style). Returns an unbind
// that restores the previous click handler and detaches.
func BindCommand(c *Command, onClick *func(), setEnabled func(bool)) (unbind func()) {
	prev := *onClick
	*onClick = func() {
		if prev != nil {
			prev()
		}
		c.Execute()
	}
	var unsub func()
	if setEnabled != nil {
		setEnabled(c.CanExecute())
		unsub = c.SubscribeCanExecuteChanged(func() { setEnabled(c.CanExecute()) })
	}
	return func() {
		*onClick = prev
		if unsub != nil {
			unsub()
		}
	}
}

// BindList projects an ObservableList[T] into a widget's []string backing slice
// (ListBox.Items, or a row source) via project, rebuilding it on every change
// and calling invalidate. Rebuild is O(n) — n is a viewport-scale list a view
// already walks each frame; the granular ListEvent is available for callers who
// want an incremental variant. Returns an unbind that detaches.
func BindList[T any](l *ObservableList[T], items *[]string, project func(T) string, invalidate func()) (unbind func()) {
	rebuild := func() {
		src := l.Slice()
		out := make([]string, len(src))
		for i, v := range src {
			out[i] = project(v)
		}
		*items = out
		if invalidate != nil {
			invalidate()
		}
	}
	rebuild()
	return l.Subscribe(func(ListEvent[T]) { rebuild() })
}
