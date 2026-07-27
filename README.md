# go-widgets/mvvm

A tiny, dependency-free **MVVM** (Model–View–ViewModel) layer for the
[go-widgets](https://github.com/go-widgets) ecosystem. Generics-only, no
reflection, 100% statement coverage.

It provides the three MVVM primitives and the binding glue to wire them to
widgets — **without importing any widget package**, so a single ViewModel drives
both the pixel [`toolkit`](https://github.com/go-widgets/toolkit) and the
terminal-cell [`tui`](https://github.com/go-widgets/tui).

| Primitive | Role |
| --- | --- |
| `Observable[T]` | a bindable **property** (Get/Set/Subscribe, skips equal values, re-entrancy-safe) |
| `Command` | a bindable **action** with `CanExecute` + `RaiseCanExecuteChanged` |
| `ObservableList[T]` | a bindable **collection** emitting granular insert/remove/replace/move/reset events |

Binding adapters reach a widget only through pointers to its value field and its
callback slot — `BindField`, `OneWay`, `BindCommand`, `BindList`.

## Why it's backend-agnostic

`go-widgets/toolkit` and `go-widgets/tui` mirror the same field names and
callback signatures per widget (`Entry.Text` + `Entry.OnChange`, `Scale.Value` +
`Scale.OnChange`, …). So a binding written against `(&w.Text, &w.OnChange)`
compiles and runs for either backend. This package therefore imports neither —
it depends only on that field/callback shape.

## Why binding is loop-free

A widget fires its change callback from inside its event handler, and leaves
direct field writes silent. So the **View→ViewModel** edge is the callback, the
**ViewModel→View** edge is a silent field write, and `Observable.Set` skips
equal values — a two-way binding can echo without recursing.

## A form in ~10 lines

```go
type FormVM struct {
	Name  *mvvm.Observable[string]
	Names *mvvm.ObservableList[string]
	Save  *mvvm.Command
}

vm := &FormVM{Name: mvvm.NewObservable(""), Names: mvvm.NewObservableList[string]()}
vm.Save = mvvm.NewCommand(
	func() { vm.Names.Append(vm.Name.Get()); vm.Name.Set("") },
	func() bool { return vm.Name.Get() != "" }, // CanExecute
)
mvvm.BindCanExecute(vm.Save, vm.Name) // Save re-greys as the name changes

// View — identical for tui except the widget types + the repaint hook.
mvvm.BindField(vm.Name, &name.Text, &name.OnChange, repaint)
mvvm.BindCommand(vm.Save, &save.OnClick, setEnabled)
mvvm.BindList(vm.Names, &list.Items, func(s string) string { return s }, repaint)
```

The **same `FormVM`** drives the pixel and the cell form verbatim.

## Backend adapters

The core package binds any widget whose value + change-callback fit
`(&field, &hook)`. Widgets with a **multi-argument** or oddly-named callback get
a small named adapter in a per-backend subpackage — the **only** packages that
import a backend:

- **`mvvm/tkbind`** (imports `toolkit`) — `BindRange` for the two-handle
  `RangeSlider` (`OnChange(low, high)`).
- **`mvvm/tuibind`** (imports `tui`) — `BindDropdown` (`OnChange(idx, value)`)
  and `BindTableSelection` (`OnSelect(row)`).

The core `mvvm` package itself still imports nothing (verified with
`go list -deps`), so a consumer who only wants observables/commands pays for no
backend.

## Status

`v0.2.0`: core + `tkbind` + `tuibind`, all 100% coverage.

## License

BSD-3-Clause — see [LICENSE](LICENSE).
