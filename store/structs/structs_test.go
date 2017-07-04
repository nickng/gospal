package structs

import (
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"

	"github.com/nickng/gospal/ssa/build"
)

var T = `package main
type T struct {
	X int
	Y struct {
		Z byte
		A string
	}
}
func main() {
	x := new(T)
	x.X = 1
	x.Y.A = ""
}`

type empty struct{}

func (empty) UniqName() string { return "_" }

func TestStruct(t *testing.T) {
	info, err := build.FromReader(strings.NewReader(T)).Default().Build()
	if err != nil {
		t.Errorf("cannot build SSA: %v", err)
	}
	pkg := info.Prog.AllPackages()[0]
	typeOnly := FromType(pkg.Type("T").Type().Underlying().(*types.Struct))
	if expect, got := 2, len(typeOnly.Fields); expect != got {
		t.Errorf("Unexpanded T should should %d field but got %d", expect, got)
	}
	typeExpanded := typeOnly.Expand()
	t.Logf("T      %s:%s", typeExpanded[0].Name(), typeExpanded[0].Type())
	t.Logf("X T.#1 %s:%s", typeExpanded[1].Name(), typeExpanded[1].Type())
	t.Logf("Y T.#2 %s:%s", typeExpanded[2].Name(), typeExpanded[2].Type())
	t.Logf("Y      %s:%s", typeExpanded[3].Name(), typeExpanded[3].Type())
	t.Logf("Z Y.#1 %s:%s", typeExpanded[4].Name(), typeExpanded[4].Type())
	t.Logf("A Y.#2 %s:%s", typeExpanded[5].Name(), typeExpanded[5].Type())
	if expect, got := 6, len(typeExpanded); expect != got {
		t.Errorf("Expanded T should have %d field but got %d", expect, got)
	}
	valOnly := New(empty{}, pkg.Func("main").Blocks[0].Instrs[0].(*ssa.Alloc))
	valExpanded := valOnly.Expand()
	t.Logf("T      %s:%s", valExpanded[0].Name(), valExpanded[0].Type())
	t.Logf("X T.#1 %s:%s", valExpanded[1].Name(), valExpanded[1].Type())
	t.Logf("Y T.#2 %s:%s", valExpanded[2].Name(), valExpanded[2].Type())
	t.Logf("Y      %s:%s", valExpanded[3].Name(), valExpanded[3].Type())
	t.Logf("Z Y.#1 %s:%s", valExpanded[4].Name(), valExpanded[4].Type())
	t.Logf("A Y.#2 %s:%s", valExpanded[5].Name(), valExpanded[5].Type())
	if valExpanded[0].Name() != "t0" {
		t.Errorf("x should be t0 but got %s", valExpanded[0].Name())
	}
	if valExpanded[1].Name() != "t0_0" {
		t.Errorf("x.X should be t0_0 but got %s", valExpanded[1].Name())
	}
	if valExpanded[2].Name() != "t0_1" {
		t.Errorf("x.X should be t0_1 but got %s", valExpanded[2].Name())
	}
	val2Only := New(empty{}, pkg.Func("main").Blocks[0].Instrs[9].(*ssa.FieldAddr))
	val2Expanded := val2Only.Expand()
	t.Logf("Y      %s:%s", val2Expanded[0].Name(), val2Expanded[0].Type())
	t.Logf("Z Y.#1 %s:%s", val2Expanded[1].Name(), val2Expanded[1].Type())
	t.Logf("A Y.#2 %s:%s", val2Expanded[2].Name(), val2Expanded[2].Type())
	if val2Expanded[0].Name() != "t2" {
		t.Errorf("x.Y should be t2 but got %s", val2Expanded[0].Name())
	}
	if val2Expanded[1].Name() != "t2_0" {
		t.Errorf("x.Y.Z should be t2_0 but got %s", val2Expanded[1].Name())
	}
	if val2Expanded[2].Name() != "t2_1" {
		t.Errorf("x.Y.A should be t2_1 but got %s", val2Expanded[2].Name())
	}
}
