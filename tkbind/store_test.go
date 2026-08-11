// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tkbind

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/go-widgets/data"
	"github.com/go-widgets/toolkit"
)

// product is the typed row the store fixtures carry.
type product struct {
	ID   string
	Name string
	Qty  int
	Cat  string
}

// productCodec (de)serialises a product to and from a data.Record.
var productCodec = data.Codec[product]{
	Encode: func(p product) data.Record {
		return data.Record{
			"id":   data.String(p.ID),
			"name": data.String(p.Name),
			"qty":  data.Int(int64(p.Qty)),
			"cat":  data.String(p.Cat),
		}
	},
	Decode: func(r data.Record) product {
		return product{
			ID:   r["id"].Str,
			Name: r["name"].Str,
			Qty:  int(r["qty"].Int),
			Cat:  r["cat"].Str,
		}
	},
}

// productSchema declares the four fields, with a Required rule on the key so a
// schema-level validation failure is reachable in a test.
var productSchema = data.Schema{Fields: []data.Field{
	{Name: "id", Kind: data.KindString, Rules: []data.Rule{data.Required("id required")}},
	{Name: "name", Kind: data.KindString},
	{Name: "qty", Kind: data.KindInt},
	{Name: "cat", Kind: data.KindString},
}}

// col indices used throughout the tests.
const (
	colID = iota
	colName
	colQty
	colCat
)

// productCols maps the four Table columns onto a product. The Name setter
// UPPER-cases its value: a committed edit that round-trips through the store
// therefore lands in the grid upper-cased, which is how a test tells a genuine
// store refresh apart from the Table writing its own raw cell.
var productCols = []TableColumnMap[product]{
	{Field: "id", Get: func(p product) string { return p.ID }},
	{Field: "name", Get: func(p product) string { return p.Name }, Set: func(p product, s string) product { p.Name = strings.ToUpper(s); return p }},
	{Field: "qty", Get: func(p product) string { return strconv.Itoa(p.Qty) }, Set: func(p product, s string) product { n, _ := strconv.Atoi(s); p.Qty = n; return p }},
	{Field: "cat", Get: func(p product) string { return p.Cat }},
}

// tableCols builds the matching toolkit.TableColumn slice (300px = 4×75, so the
// table fits its bounds and never scrolls horizontally — columnAt stays simple).
func tableCols() []toolkit.TableColumn {
	return []toolkit.TableColumn{
		{Title: "ID", Width: 75},
		{Title: "Name", Width: 75, Editable: true, Validate: toolkit.Required("name required")},
		{Title: "Qty", Width: 75, Sortable: true, Editable: true},
		{Title: "Cat", Width: 75, Sortable: true},
	}
}

// errBoom is the sentinel a failProxy returns.
var errBoom = errors.New("boom")

// failProxy wraps a MemoryProxy and can be told to fail any of the three Proxy
// operations. It doubles as a SECOND Proxy implementation, so a green run over
// it proves the binding is proxy-agnostic (it reaches data only through the
// Store/Proxy seam) exactly as the memory and gRPC proxies are interchangeable.
type failProxy struct {
	inner                        *data.MemoryProxy
	failList, failQuery, failMut bool
}

func (f *failProxy) List(ctx context.Context) ([]data.Record, error) {
	if f.failList {
		return nil, errBoom
	}
	return f.inner.List(ctx)
}

func (f *failProxy) Query(ctx context.Context, q data.Query) (data.View, error) {
	if f.failQuery {
		return data.View{}, errBoom
	}
	return f.inner.Query(ctx, q)
}

func (f *failProxy) Mutate(ctx context.Context, m data.Mutation) error {
	if f.failMut {
		return errBoom
	}
	return f.inner.Mutate(ctx, m)
}

// newFailProxy seeds a failProxy with the given products.
func newFailProxy(t *testing.T, seed ...product) *failProxy {
	t.Helper()
	recs := make([]data.Record, len(seed))
	for i, p := range seed {
		recs[i] = productCodec.Encode(p)
	}
	mp, err := data.NewMemoryProxy(productSchema, "id", recs...)
	if err != nil {
		t.Fatalf("NewMemoryProxy: %v", err)
	}
	return &failProxy{inner: mp}
}

// fixture builds a loaded store + table wired by BindStore, returning the pieces
// plus an *int counting BindStore repaints.
type fixture struct {
	proxy  *failProxy
	store  *data.Store[product]
	table  *toolkit.Table
	paints int
	unbind func()
}

func newFixture(t *testing.T, seed ...product) *fixture {
	t.Helper()
	fp := newFailProxy(t, seed...)
	st := data.NewStore(data.Proxy(fp), productCodec)
	if _, err := st.Load(context.Background()); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	tbl := toolkit.NewTable(tableCols(), nil)
	tbl.SetBounds(toolkit.Rect{X: 0, Y: 0, W: 300, H: 240})
	fx := &fixture{proxy: fp, store: st, table: tbl}
	fx.unbind = BindStore(st, tbl, productCols, func() { fx.paints++ })
	return fx
}

// wantRows asserts the table's whole row model equals the projection of want.
func wantRows(t *testing.T, tbl *toolkit.Table, want []product) {
	t.Helper()
	if len(tbl.Rows) != len(want) {
		t.Fatalf("row count = %d, want %d (%v)", len(tbl.Rows), len(want), tbl.Rows)
	}
	for i, p := range want {
		exp := []string{p.ID, p.Name, strconv.Itoa(p.Qty), p.Cat}
		for j, cell := range exp {
			if tbl.Rows[i][j] != cell {
				t.Fatalf("row %d col %d = %q, want %q", i, j, tbl.Rows[i][j], cell)
			}
		}
	}
}

// fakeEditor is a controllable CellEditor so a test can drive the Table's real
// commit path (validation included) with an exact value.
type fakeEditor struct {
	toolkit.Base
	value  string
	submit func()
}

func (e *fakeEditor) CellValue() string      { return e.value }
func (e *fakeEditor) SetCellValue(s string)  { e.value = s }
func (e *fakeEditor) OnCellSubmit(fn func()) { e.submit = fn }
func (e *fakeEditor) Focus(bool)             {}

// editVia opens an editor on (row,col), sets it to value, and commits through
// the Table so the column's Validate rule and OnCellEdit/OnCellEditRejected all
// run for real. It returns the editor so a caller can inspect a rejected edit.
func editVia(tbl *toolkit.Table, row, col int, value string) *fakeEditor {
	ed := &fakeEditor{}
	tbl.Columns[col].Editor = func() toolkit.CellEditor { return ed }
	tbl.BeginEdit(row, col)
	ed.value = value
	tbl.CommitEdit()
	return ed
}

func ctx() context.Context { return context.Background() }

// --- BindStore ------------------------------------------------------------

func TestBindStore_ProjectsAndReacts(t *testing.T) {
	fx := newFixture(t,
		product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		product{ID: "b", Name: "Bread", Qty: 1, Cat: "bakery"},
	)
	wantRows(t, fx.table, []product{
		{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		{ID: "b", Name: "Bread", Qty: 1, Cat: "bakery"},
	})
	if fx.paints != 1 {
		t.Fatalf("initial paints = %d, want 1", fx.paints)
	}

	// A store mutation reloads Items and must rebuild the table's rows.
	if err := fx.store.Add(ctx(), product{ID: "c", Name: "Cream", Qty: 5, Cat: "dairy"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantRows(t, fx.table, []product{
		{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		{ID: "b", Name: "Bread", Qty: 1, Cat: "bakery"},
		{ID: "c", Name: "Cream", Qty: 5, Cat: "dairy"},
	})
	// Each reload is a Clear+Append on the store's Items, so it repaints twice;
	// the exact count is data's concern — assert only that a refresh happened.
	if fx.paints <= 1 {
		t.Fatalf("paints after Add = %d, want a refresh (>1)", fx.paints)
	}
}

func TestBindStore_NilInvalidateAndUnbind(t *testing.T) {
	fp := newFailProxy(t, product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"})
	st := data.NewStore(data.Proxy(fp), productCodec)
	if _, err := st.Load(ctx()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	tbl := toolkit.NewTable(tableCols(), nil)
	unbind := BindStore(st, tbl, productCols, nil) // nil invalidate must not panic
	wantRows(t, tbl, []product{{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"}})

	unbind()
	// After unbind, a store change no longer touches the table.
	if err := st.Add(ctx(), product{ID: "z", Name: "Zebra", Qty: 9}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantRows(t, tbl, []product{{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"}})
}

// --- BindTable: edit round-trip ------------------------------------------

func TestBindTable_EditRoundTrip(t *testing.T) {
	fx := newFixture(t,
		product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		product{ID: "b", Name: "Bread", Qty: 1, Cat: "bakery"},
	)
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(error) {
		t.Fatalf("unexpected error")
	})
	defer b.Unbind()

	before := fx.paints
	// Edit row 0's Name to "kiwi"; the Set upper-cases it, so a real store
	// refresh lands "KIWI" in the grid, not the raw "kiwi" the Table wrote.
	editVia(fx.table, 0, colName, "kiwi")

	// 1. The backing Record was mutated THROUGH the proxy.
	recs, err := fx.proxy.List(ctx())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got string
	for _, r := range recs {
		if r["id"].Str == "a" {
			got = r["name"].Str
		}
	}
	if got != "KIWI" {
		t.Fatalf("record name = %q, want KIWI (store not mutated through proxy)", got)
	}
	// 2. The grid refreshed from the reloaded store (upper-cased projection).
	wantRows(t, fx.table, []product{
		{ID: "a", Name: "KIWI", Qty: 3, Cat: "fruit"},
		{ID: "b", Name: "Bread", Qty: 1, Cat: "bakery"},
	})
	// 3. The refresh went through BindStore (a repaint fired).
	if fx.paints <= before {
		t.Fatalf("paints did not advance on refresh: before=%d after=%d", before, fx.paints)
	}
}

func TestBindTable_EditRejectedByTableValidation(t *testing.T) {
	fx := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"})
	var gotErr error
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(e error) { gotErr = e })
	defer b.Unbind()

	// An empty Name fails the column's Required rule: OnCellEdit never fires, so
	// the store is untouched; OnCellEditRejected reaches onErr.
	ed := editVia(fx.table, 0, colName, "")
	if gotErr == nil {
		t.Fatalf("expected a rejection error through onErr")
	}
	if fx.table.EditError() == nil {
		t.Fatalf("table should report the edit error and keep the editor open")
	}
	_ = ed
	// The record is unchanged: no mutation reached the proxy.
	recs, _ := fx.proxy.List(ctx())
	if recs[0]["name"].Str != "Apple" {
		t.Fatalf("record name = %q, want Apple (rejected edit must not mutate)", recs[0]["name"].Str)
	}
}

func TestBindTable_EditUpdateErrorSurfaces(t *testing.T) {
	fx := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"})
	var gotErr error
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(e error) { gotErr = e })
	defer b.Unbind()

	fx.proxy.failMut = true // the proxy now rejects the update
	editVia(fx.table, 0, colName, "kiwi")
	if !errors.Is(gotErr, errBoom) {
		t.Fatalf("update error = %v, want errBoom", gotErr)
	}
}

// --- BindTable: sort ------------------------------------------------------

func TestBindTable_SortViaHeaderClick(t *testing.T) {
	fx := newFixture(t,
		product{ID: "a", Name: "Apple", Qty: 3, Cat: "x"},
		product{ID: "b", Name: "Bread", Qty: 1, Cat: "x"},
		product{ID: "c", Name: "Cream", Qty: 2, Cat: "x"},
	)
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(err error) {
		t.Fatalf("unexpected error: %v", err)
	})
	defer b.Unbind()

	// Click the Qty header (col 2 spans x∈[150,225), header y<24) — a real event.
	fx.table.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: 160, Y: 5})

	// The Table flipped its own indicator...
	if fx.table.SortColumn != colQty || !fx.table.SortAsc {
		t.Fatalf("indicator = (col %d, asc %v), want (2, true)", fx.table.SortColumn, fx.table.SortAsc)
	}
	// ...the binding set the query...
	q := fx.store.Query()
	if len(q.Sorts) != 1 || q.Sorts[0].Field != "qty" || q.Sorts[0].Desc {
		t.Fatalf("query sorts = %+v, want one asc sort on qty", q.Sorts)
	}
	// ...and the reload delivered rows ordered by qty ascending.
	wantRows(t, fx.table, []product{
		{ID: "b", Name: "Bread", Qty: 1, Cat: "x"},
		{ID: "c", Name: "Cream", Qty: 2, Cat: "x"},
		{ID: "a", Name: "Apple", Qty: 3, Cat: "x"},
	})

	// A second click on the same column toggles to descending.
	fx.table.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: 160, Y: 5})
	if q := fx.store.Query(); !q.Sorts[0].Desc {
		t.Fatalf("second click should sort descending, got %+v", q.Sorts)
	}
	wantRows(t, fx.table, []product{
		{ID: "a", Name: "Apple", Qty: 3, Cat: "x"},
		{ID: "c", Name: "Cream", Qty: 2, Cat: "x"},
		{ID: "b", Name: "Bread", Qty: 1, Cat: "x"},
	})
}

func TestBindTable_SortByOutOfRangeIgnored(t *testing.T) {
	fx := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3})
	b := BindTable(fx.store, fx.table, productCols, ctx(), nil)
	defer b.Unbind()
	b.SortBy(-1, true)
	b.SortBy(99, true)
	if len(fx.store.Query().Sorts) != 0 {
		t.Fatalf("out-of-range SortBy must not touch the query: %+v", fx.store.Query().Sorts)
	}
}

// --- BindTable: group -----------------------------------------------------

func TestBindTable_GroupAndUngroup(t *testing.T) {
	fx := newFixture(t,
		product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		product{ID: "d", Name: "Donut", Qty: 4, Cat: "bakery"},
		product{ID: "e", Name: "Egg", Qty: 6, Cat: "fruit"},
	)
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(err error) {
		t.Fatalf("unexpected error: %v", err)
	})
	defer b.Unbind()

	b.GroupBy(colCat)
	if fx.store.Query().GroupBy != "cat" {
		t.Fatalf("query GroupBy = %q, want cat", fx.store.Query().GroupBy)
	}
	if fx.table.GroupBy != colCat {
		t.Fatalf("table GroupBy = %d, want %d", fx.table.GroupBy, colCat)
	}
	// Grouped rows flatten in first-appearance group order: the fruit bucket
	// (seeded first, via "a") then the bakery bucket ("d").
	wantRows(t, fx.table, []product{
		{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"},
		{ID: "e", Name: "Egg", Qty: 6, Cat: "fruit"},
		{ID: "d", Name: "Donut", Qty: 4, Cat: "bakery"},
	})

	// An out-of-range column ungroups.
	b.GroupBy(-1)
	if fx.store.Query().GroupBy != "" || fx.table.GroupBy != -1 {
		t.Fatalf("ungroup failed: query=%q table=%d", fx.store.Query().GroupBy, fx.table.GroupBy)
	}
}

// --- BindTable: commit guards --------------------------------------------

func TestBindTable_CommitGuards(t *testing.T) {
	fx := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3, Cat: "fruit"})
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(error) {
		t.Fatalf("guarded commit must not error")
	})
	defer b.Unbind()

	// Drive the wired OnCellEdit closure directly for the guarded shapes the
	// Table itself would never emit but a host might, covering every early exit.
	fx.table.OnCellEdit(0, colID, "x")    // column with a nil Set (read-only)
	fx.table.OnCellEdit(0, -1, "x")       // column out of range (low)
	fx.table.OnCellEdit(0, 99, "x")       // column out of range (high)
	fx.table.OnCellEdit(-1, colName, "x") // row out of range (low)
	fx.table.OnCellEdit(99, colName, "x") // row out of range (high)

	recs, _ := fx.proxy.List(ctx())
	if recs[0]["name"].Str != "Apple" || recs[0]["id"].Str != "a" {
		t.Fatalf("guarded commits mutated the store: %+v", recs[0])
	}
}

// --- BindTable: reload error + nil onErr ---------------------------------

func TestBindTable_ReloadErrorAndNilOnErr(t *testing.T) {
	fx := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3})
	var gotErr error
	b := BindTable(fx.store, fx.table, productCols, ctx(), func(e error) { gotErr = e })
	defer b.Unbind()

	fx.proxy.failQuery = true // reload will now fail
	b.SortBy(colQty, true)
	if !errors.Is(gotErr, errBoom) {
		t.Fatalf("reload error = %v, want errBoom", gotErr)
	}

	// A binding with a nil onErr must swallow the same failure without panicking.
	fx2 := newFixture(t, product{ID: "a", Name: "Apple", Qty: 3})
	b2 := BindTable(fx2.store, fx2.table, productCols, ctx(), nil)
	defer b2.Unbind()
	fx2.proxy.failQuery = true
	b2.SortBy(colQty, true) // fail(nil) branch — no panic, no observer
}

// --- BindTable: callback composition + Unbind ----------------------------

func TestBindTable_ComposesPriorCallbacksAndUnbinds(t *testing.T) {
	fx := newFixture(t,
		product{ID: "a", Name: "Apple", Qty: 2, Cat: "x"},
		product{ID: "b", Name: "Bread", Qty: 1, Cat: "x"},
	)
	var sortSeen, editSeen, rejectSeen bool
	fx.table.OnSort = func(int, bool) { sortSeen = true }
	fx.table.OnCellEdit = func(int, int, string) { editSeen = true }
	fx.table.OnCellEditRejected = func(int, int, string, error) { rejectSeen = true }

	b := BindTable(fx.store, fx.table, productCols, ctx(), func(error) {})

	// A header click runs the prior OnSort AND the store wiring.
	fx.table.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: 160, Y: 5})
	if !sortSeen {
		t.Fatalf("prior OnSort was not composed")
	}
	if len(fx.store.Query().Sorts) == 0 {
		t.Fatalf("store wiring did not run after prior OnSort")
	}
	// A valid edit runs the prior OnCellEdit; an invalid one the prior reject.
	editVia(fx.table, 0, colName, "kiwi")
	if !editSeen {
		t.Fatalf("prior OnCellEdit was not composed")
	}
	editVia(fx.table, 0, colName, "")
	if !rejectSeen {
		t.Fatalf("prior OnCellEditRejected was not composed")
	}

	// Unbind restores the exact prior callbacks (fresh flags prove identity).
	b.Unbind()
	sortSeen, editSeen, rejectSeen = false, false, false
	fx.table.OnSort(0, true)
	fx.table.OnCellEdit(0, 0, "")
	fx.table.OnCellEditRejected(0, 0, "", nil)
	if !sortSeen || !editSeen || !rejectSeen {
		t.Fatalf("Unbind did not restore prior callbacks: %v %v %v", sortSeen, editSeen, rejectSeen)
	}
	// And the store wiring is detached: the restored OnSort must not touch the query.
	q0 := fx.store.Query()
	fx.table.OnSort(colCat, true)
	if len(fx.store.Query().Sorts) != len(q0.Sorts) {
		t.Fatalf("store wiring still active after Unbind")
	}
}
