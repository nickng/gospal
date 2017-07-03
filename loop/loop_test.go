package loop

import (
	"strings"
	"testing"

	"github.com/nickng/gospal/block"
	"github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/ssa/build"
	gossa "golang.org/x/tools/go/ssa"
)

type loopDetector struct {
	d *Detector
}

func (ld loopDetector) detect(from, to *gossa.BasicBlock) {
	if from != nil {
		ld.d.Detect(from, to)
		for _, instr := range to.Instrs {
			switch instr := instr.(type) {
			case *gossa.Phi:
				ld.d.ExtractIndex(instr)
			case *gossa.If:
				ld.d.ExtractCond(instr)
			}
		}
	}
}

func TestSimpleLoop(t *testing.T) {
	loop := "t2 = 0; (t2<10); t2 = t2 + 1"
	src := `package main
	func yes(i int) bool { return true }
	func main() {
		for i := 0; i < 10; i++ {
			if i := 1; yes(i) { // this does not clash with i of outer scope.
			}
		}
	}`
	info, err := build.FromReader(strings.NewReader(src)).Default().Build()
	if err != nil {
		t.Error("cannot build SSA:", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Error("Cannot find main package:", err)
	}
	ld := loopDetector{d: NewDetector()}
	for _, main := range mains {
		block.TraverseEdges(main.Func("main"), ld.detect)
		if expect, got := loop, ld.d.ForLoopAt(main.Func("main").Blocks[1]).String(); expect != got {
			t.Errorf("Loop not extracted correctly, want:\n%s\ngot:\n%s\n",
				expect, got)
		}
		t.Logf("Extracted loop: %s", ld.d.ForLoopAt(main.Func("main").Blocks[1]).String())
	}
}

func TestNestedLoop(t *testing.T) {
	loopi := "t0 = 0; (t0<10); t0 = t0 + 1"
	loopj := "t4 = 1; (t4<9); t4 = t4 + 2"
	src := `package main
	func yes(i int) bool { return true }
	func main() {
		for i := 0; i < 10; i++ {
			for j := 1; j < 9; j+=2 {
			}
		}
	}
	`
	info, err := build.FromReader(strings.NewReader(src)).Default().Build()
	if err != nil {
		t.Error("cannot build SSA:", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Error("Cannot find main package:", err)
	}
	ld := loopDetector{d: NewDetector()}
	for _, main := range mains {
		block.TraverseEdges(main.Func("main"), ld.detect)
		if expect, got := loopi, ld.d.ForLoopAt(main.Func("main").Blocks[1]).String(); expect != got {
			t.Errorf("Loop not extracted correctly, want:\n%s\ngot:\n%s\n",
				expect, got)
		}
		if expect, got := loopj, ld.d.ForLoopAt(main.Func("main").Blocks[4]).String(); expect != got {
			t.Errorf("Loop not extracted correctly, want:\n%s\ngot:\n%s\n",
				expect, got)
		}
		t.Logf("Extracted loop-i: %s", ld.d.ForLoopAt(main.Func("main").Blocks[1]).String())
		t.Logf("Extracted loop-j: %s", ld.d.ForLoopAt(main.Func("main").Blocks[4]).String())
	}
}

func TestShortCircuitedLoop(t *testing.T) {
	loop := "t1 = 0; ((t1<10) && ((t1%2)==0)); t1 = t1 + 1"
	src := `package main
	func main() {
		for i := 0; i<10 && i%2 == 0; i++ {
		}
	}
	`
	info, err := build.FromReader(strings.NewReader(src)).Default().Build()
	if err != nil {
		t.Error("cannot build SSA:", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Error("Cannot find main package:", err)
	}
	ld := loopDetector{d: NewDetector()}
	for _, main := range mains {
		block.TraverseEdges(main.Func("main"), ld.detect)
		if expect, got := loop, ld.d.ForLoopAt(main.Func("main").Blocks[1]).String(); expect != got {
			t.Errorf("Loop not extracted correctly, want:\n%s\ngot:\n%s\n",
				expect, got)
		}
	}
}

func TestNotLoop(t *testing.T) {
	src := `package main
	func main() {
		for i := 0; ; i++ {
		}
	}`
	info, err := build.FromReader(strings.NewReader(src)).Default().Build()
	if err != nil {
		t.Error("cannot build SSA:", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Error("Cannot find main package:", err)
	}
	ld := loopDetector{d: NewDetector()}
	for _, main := range mains {
		if info := ld.d.ForLoopAt(main.Func("main").Blocks[1]); info != nil && info.ParamsOK() {
			t.Error("Not a complete loop but detected as complete loop:",
				ld.d.ForLoopAt(main.Func("main").Blocks[1]).String())
		}
	}
}
