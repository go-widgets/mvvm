// Copyright (c) 2026 the go-widgets/mvvm authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mvvm_test

import (
	"fmt"

	"github.com/go-widgets/mvvm"
)

// widget stands in for a real toolkit.Entry / tui.Entry — the same field+hook
// surface both backends expose, so a binding written against pointers works for
// either without this package importing a backend.
type widget struct {
	Text     string
	OnChange func(string)
}

func ExampleObservable() {
	name := mvvm.NewObservable("")
	name.Subscribe(func(v string) { fmt.Println("name is now", v) })
	name.Set("Ada")
	name.Set("Ada") // equal → no notification
	name.Set("Bob")
	// Output:
	// name is now Ada
	// name is now Bob
}

func ExampleCommand() {
	name := mvvm.NewObservable("")
	save := mvvm.NewCommand(
		func() { fmt.Println("saved", name.Get()) },
		func() bool { return name.Get() != "" }, // CanExecute
	)
	save.Execute() // gated off (name empty) → nothing happens
	name.Set("Ada")
	save.Execute()
	// Output:
	// saved Ada
}

func ExampleObservableList() {
	todos := mvvm.NewObservableList[string]("buy milk")
	todos.Subscribe(func(e mvvm.ListEvent[string]) {
		fmt.Printf("change kind=%d at %d\n", e.Kind, e.Index)
	})
	todos.Append("walk dog")
	todos.RemoveAt(0)
	fmt.Println(todos.Slice())
	// Output:
	// change kind=0 at 1
	// change kind=1 at 0
	// [walk dog]
}

// ExampleBindField shows the two-way binding an app's View layer sets up: the
// ViewModel's observable and a widget's (Text, OnChange) drive each other, with
// no backend import.
func ExampleBindField() {
	vmName := mvvm.NewObservable("seed")
	w := &widget{}
	mvvm.BindField(vmName, &w.Text, &w.OnChange, nil)

	fmt.Println("seeded:", w.Text) // View seeded from the ViewModel
	w.OnChange("typed by user")    // View → ViewModel
	fmt.Println("vm:", vmName.Get())
	vmName.Set("set by code") // ViewModel → View
	fmt.Println("view:", w.Text)
	// Output:
	// seeded: seed
	// vm: typed by user
	// view: set by code
}
