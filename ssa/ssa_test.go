package ssa_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/ssa/build"
)

// This tests basic build.
func TestBuild(t *testing.T) {
	s := `package main
	import "fmt"
	func main() {
		fmt.Println("Hello World")
	}`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	if info.Prog == nil {
		t.Errorf("SSA Program missing")
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("cannot find main packages: %v", err)
	}
	for _, main := range mains {
		if main.Func("main") == nil {
			t.Error("expects main.main() but not found")
		}
	}
}

// This tests building with non-main package.
func TestBuildNonMainPkg(t *testing.T) {
	s := `package pkg
	import "fmt"
	func main() {
		fmt.Println("Hello World")
	}`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if _, err = ssa.MainPkgs(info.Prog, false); err != ssa.ErrNoMainPkgs {
		t.Errorf("unexpected main package")
	}
}

// This tests building of callgraph.
func TestCallGraph(t *testing.T) {
	s := `package main
	import "fmt"
	func main() {
		foo("Hello")
	}
	func foo(s string) {
		fmt.Println(s, "World")
	}
	func bar() {
		fmt.Println("doesn't reach here")
	}`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	if info.Prog == nil {
		t.Errorf("SSA Program missing")
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("cannot find main packages: %v", err)
	}
	for _, main := range mains {
		if main.Func("main") == nil {
			t.Error("expects main.main() but not found")
		}
	}
	graph, err := info.BuildCallGraph("pta", false)
	if err != nil {
		t.Errorf("build callgraph failed: %v", err)
	}
	fns, err := graph.UsedFunctions()
	if err != nil {
		t.Errorf("cannot filter unused functions in callgraph: %v", err)
	}
	for _, fn := range fns {
		if fn.Pkg.Pkg.Name() == "main" {
			if fn.Name() != "foo" && fn.Name() != "main" && fn.Name() != "init" {
				t.Errorf("expecting main.{init, main, foo}, but got main.%s", fn.Name())
			}
		}
	}
}

// This tests building of callgraph and retrieving of all functions in callgraph.
func TestCallGraphAllFunc(t *testing.T) {
	s := `package main
	import "fmt"
	func main() {
		foo("Hello")
	}
	func foo(s string) {
		fmt.Println(s, "World")
	}
	func bar() {
		fmt.Println("doesn't reach here")
	}`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	if info.Prog == nil {
		t.Errorf("SSA Program missing")
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("cannot find main packages: %v", err)
	}
	for _, main := range mains {
		if main.Func("main") == nil {
			t.Error("expects main.main() but not found")
		}
	}
	graph, err := info.BuildCallGraph("pta", false)
	if err != nil {
		t.Errorf("build callgraph failed: %v", err)
	}

	allFuncs, err := graph.AllFunctions()
	if err != nil {
		t.Errorf("cannot get functions in callgraph: %v", err)
	}
	usedFuncs, err := graph.UsedFunctions()
	if err != nil {
		t.Errorf("cannot filter unused functions in callgraph: %v", err)
	}
	if len(allFuncs) < len(usedFuncs) {
		t.Errorf("callgraph has %d functions, %d are used. Expect used < all",
			len(allFuncs), len(usedFuncs))
	}
}

func ExampleInfo_WriteTo() {
	s := `package main
	func main() { }`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if err != nil {
		log.Fatalf("SSA build failed: %v", err)
	}
	var buf bytes.Buffer
	info.WriteTo(&buf)
	fmt.Println(buf.String())
	// output:
	// # Name: main.init
	// # Package: main
	// # Synthetic: package initializer
	// func init():
	// 0:                                                                entry P:0 S:0
	// 	return
	//
	// # Name: main.main
	// # Package: main
	// # Location: tmp:2:7
	// func main():
	// 0:                                                                entry P:0 S:0
	// 	return
}

func ExampleCallGraph_WriteGraphviz() {
	s := `package main
	func main() { }`

	conf := build.FromReader(strings.NewReader(s))
	info, err := conf.Build()
	if err != nil {
		log.Fatalf("SSA build failed: %v", err)
	}
	var buf bytes.Buffer
	cg, err := info.BuildCallGraph("pta", false) // Pointer analysis, no tests.
	if err != nil {
		log.Fatalf("Cannot build callgraph: %v", err)
	}
	cg.WriteGraphviz(&buf)
	fmt.Println(buf.String())
	// output:
	// digraph callgraph {
	//   "<root>" -> "main.init"
	//   "<root>" -> "main.main"
	// }
}
