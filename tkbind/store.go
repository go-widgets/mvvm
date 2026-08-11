// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"context"

	"github.com/go-widgets/data"
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// TableColumnMap maps one toolkit.Table column onto the typed row R that a
// data.Store yields. It is the projection between the grid's string cell model
// and the store's typed records:
//
//   - Field names the data.Record field the column reflects. It drives the query
//     side of a BindTable: a header-click sort orders by this field and a group
//     buckets by it. It may be empty for a column that is display-only (a derived
//     or composed cell that no single record field backs).
//   - Get renders the cell string for a row. It must be non-nil.
//   - Set, when non-nil, folds an edited cell string back into the row and
//     returns the updated row, so a committed edit on this column can be pushed
//     to the store. A nil Set marks the column read-only through the binding
//     (an edit on it is ignored even if the Table itself allows editing).
//
// The slice a caller passes to BindStore / BindTable is in the Table's column
// order: the i-th map describes the i-th toolkit.TableColumn.
type TableColumnMap[R any] struct {
	Field string
	Get   func(R) string
	Set   func(row R, cell string) R
}

// BindStore projects a data.Store's decoded rows into a toolkit.Table's Rows and
// keeps the two in sync. On every change to the store's observable item list — a
// Load, an Add/Update/Delete, or any external mutation that reloads it — it
// rebuilds table.Rows through the column maps' Get and calls invalidate. It
// mirrors mvvm.BindList's full-rebuild shape, producing a [][]string row model
// from the typed rows (grouped queries flatten into the store's Items in query
// order, so a grouped store still projects to a flat, ordered row model that the
// Table's own GroupBy then renders with headers).
//
// It reads the store only through its observable Items list, so it is oblivious
// to whether the store's proxy is the in-process MemoryProxy or the remote
// grpcproxy client — the binding behaves identically native and in wasm. Returns
// an unbind that detaches the list subscription.
func BindStore[R any](store *data.Store[R], t *toolkit.Table, cols []TableColumnMap[R], invalidate func()) (unbind func()) {
	rebuild := func() {
		src := store.Items().Slice()
		rows := make([][]string, len(src))
		for i, r := range src {
			cells := make([]string, len(cols))
			for j, c := range cols {
				cells[j] = c.Get(r)
			}
			rows[i] = cells
		}
		t.Rows = rows
		if invalidate != nil {
			invalidate()
		}
	}
	rebuild()
	return store.Items().Subscribe(func(mvvm.ListEvent[R]) { rebuild() })
}

// TableBinding is the live wiring of a toolkit.Table's sort/group/edit
// interactions onto a data.Store returned by BindTable. Sort and edit flow
// automatically through the Table's own callbacks; grouping — which the Table
// exposes as host-set state with no event — is driven by the GroupBy method.
// Call Unbind to detach the wired callbacks. It is not safe for concurrent use;
// like the rest of the toolkit it expects to run on the single UI goroutine.
type TableBinding[R any] struct {
	store      *data.Store[R]
	table      *toolkit.Table
	cols       []TableColumnMap[R]
	ctx        context.Context
	onErr      func(error)
	prevSort   func(col int, ascending bool)
	prevEdit   func(row, col int, value string)
	prevReject func(row, col int, value string, err error)
}

// BindTable wires a toolkit.Table's interactions back onto a data.Store's Query
// and records, so the user driving the grid drives the headless data spine:
//
//   - a Sortable header click replaces Query.Sorts with the clicked column's
//     field (ascending as the Table reports it) and reloads. The Table has
//     already flipped its own ▲/▼ indicator before firing, so the two stay
//     consistent.
//   - a committed cell edit — which has already passed the column's Validate
//     rule, since a rejected edit fires OnCellEditRejected and never reaches
//     OnCellEdit — folds the new value into the row through the column map's Set
//     and Updates the store, mutating the backing Record through the proxy. A
//     column whose map has a nil Set is left read-only.
//   - grouping is applied via GroupBy(col), which sets Query.GroupBy to the
//     column's field and the Table's own GroupBy so headers render.
//
// Every reload flows back through the store's observable Items list, so pair
// BindTable with a BindStore on the same Table for the rows to refresh. onErr,
// when non-nil, receives any proxy or reload error, plus a rejected edit's
// validation error. Because it reaches the data only through the Store/Proxy, the
// wiring is identical whether the proxy is local (MemoryProxy) or remote
// (grpcproxy) — native or wasm. It composes with any callbacks the Table already
// carried: those run first, then the store wiring. Returns the *TableBinding.
//
// Refresh after a reload is driven through BindStore's Items subscription, so
// there is no invalidate parameter here — pair this with a BindStore on the same
// Table and pass the repaint hook there.
func BindTable[R any](store *data.Store[R], t *toolkit.Table, cols []TableColumnMap[R], ctx context.Context, onErr func(error)) *TableBinding[R] {
	b := &TableBinding[R]{store: store, table: t, cols: cols, ctx: ctx, onErr: onErr}

	b.prevSort = t.OnSort
	t.OnSort = func(col int, ascending bool) {
		if b.prevSort != nil {
			b.prevSort(col, ascending)
		}
		b.SortBy(col, ascending)
	}

	b.prevEdit = t.OnCellEdit
	t.OnCellEdit = func(row, col int, value string) {
		if b.prevEdit != nil {
			b.prevEdit(row, col, value)
		}
		b.commit(row, col, value)
	}

	b.prevReject = t.OnCellEditRejected
	t.OnCellEditRejected = func(row, col int, value string, err error) {
		if b.prevReject != nil {
			b.prevReject(row, col, value, err)
		}
		b.fail(err)
	}

	return b
}

// SortBy sets the store's query to order by the given column's field (descending
// when not ascending) and reloads. It is what the wired OnSort calls; a host may
// also call it directly. An out-of-range column is ignored.
func (b *TableBinding[R]) SortBy(col int, ascending bool) {
	if col < 0 || col >= len(b.cols) {
		return
	}
	q := b.store.Query()
	q.Sorts = []data.Sort{{Field: b.cols[col].Field, Desc: !ascending}}
	b.store.SetQuery(q)
	b.reload()
}

// GroupBy groups the store's query by the given column's field and turns on the
// Table's own grouping for that column so it renders group headers, then
// reloads. Passing an out-of-range column ungroups: it clears Query.GroupBy and
// the Table's GroupBy and reloads. Grouping is host-driven (the Table emits no
// group event), so this is a method rather than an automatic callback.
func (b *TableBinding[R]) GroupBy(col int) {
	q := b.store.Query()
	if col < 0 || col >= len(b.cols) {
		q.GroupBy = ""
		b.table.GroupBy = -1
	} else {
		q.GroupBy = b.cols[col].Field
		b.table.GroupBy = col
	}
	b.store.SetQuery(q)
	b.reload()
}

// commit pushes a committed cell edit onto the store. The value has already
// passed the column's Table-side Validate rule (an invalid edit never reaches
// here), so this only maps the string back onto the row and Updates the store,
// which re-validates against the schema at the proxy and reloads on success.
func (b *TableBinding[R]) commit(row, col int, value string) {
	if col < 0 || col >= len(b.cols) || b.cols[col].Set == nil {
		return
	}
	if row < 0 || row >= b.store.Items().Len() {
		return
	}
	updated := b.cols[col].Set(b.store.Items().At(row), value)
	if err := b.store.Update(b.ctx, updated); err != nil {
		b.fail(err)
	}
}

// reload re-runs the store's current query, surfacing any error through onErr.
// The successful path clears and refills the store's Items, whose subscription
// (see BindStore) rebuilds the Table's rows.
func (b *TableBinding[R]) reload() {
	if _, err := b.store.Load(b.ctx); err != nil {
		b.fail(err)
	}
}

// fail reports an error to onErr when one is set.
func (b *TableBinding[R]) fail(err error) {
	if b.onErr != nil {
		b.onErr(err)
	}
}

// Unbind restores the Table's callbacks to whatever they were before BindTable
// wired them, detaching the store wiring.
func (b *TableBinding[R]) Unbind() {
	b.table.OnSort = b.prevSort
	b.table.OnCellEdit = b.prevEdit
	b.table.OnCellEditRejected = b.prevReject
}
