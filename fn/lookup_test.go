package fn

import (
	gssa "github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/ssa/build"
	"golang.org/x/tools/go/ssa"
	"testing"
)

// Tests lookup of interface.
func TestLookupInterface(t *testing.T) {
	info, err := build.FromFiles("testdata/iface.go").Default().Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := gssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("no main package: %v", err)
	}
	// invoke call statement
	call := mains[0].Func("main").Blocks[0].Instrs[6]
	c, ok := call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err := LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if fn.Synthetic != "" {
		t.Errorf("Implementation is not concrete %v: %s", fn, fn.Synthetic)
	}
	if expect, got := "*main.t", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Interface lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
}

// Tests lookup of interface with value implementation.
func TestLookupInterface2(t *testing.T) {
	info, err := build.FromFiles("testdata/iface2.go").Default().Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := gssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("no main package: %v", err)
	}
	// invoke call statement
	call := mains[0].Func("main").Blocks[0].Instrs[7]
	c, ok := call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err := LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if fn.Synthetic != "" {
		fn = FindConcrete(info.Prog, fn)
	}
	if fn.Synthetic != "" {
		t.Errorf("Implementation is not concrete %v: %s", fn, fn.Synthetic)
	}
	if expect, got := "main.t", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Interface 2 lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
}

// Tests lookup of type asserted interface.
func TestLookupAssert(t *testing.T) {
	info, err := build.FromFiles("testdata/assert.go").Default().Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := gssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("no main package: %v", err)
	}
	// invoke call statement
	call := mains[0].Func("main").Blocks[0].Instrs[10]
	c, ok := call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err := LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if expect, got := "*main.u", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Asserted lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
}

func TestReturnValue(t *testing.T) {
	info, err := build.FromFiles("testdata/retval.go").Default().Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := gssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("no main package: %v", err)
	}
	// invoke call statement
	call := mains[0].Func("main").Blocks[0].Instrs[7]
	c, ok := call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err := LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if expect, got := "*main.t", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Returned interface lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
}

func TestChainedCall(t *testing.T) {
	info, err := build.FromFiles("testdata/chained.go").Default().Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := gssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("no main package: %v", err)
	}
	// invoke call statement
	call := mains[0].Func("main").Blocks[0].Instrs[7]
	c, ok := call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err := LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if expect, got := "*main.t", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Chained #1 lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
	t.Logf("%v has type %v", c.Call.Value.Name(), fn.String())
	// invoke call statement
	call = mains[0].Func("main").Blocks[0].Instrs[9]
	c, ok = call.(*ssa.Call)
	if !ok {
		t.Errorf("Expecting an invoke call: %v", call)
	}
	t.Logf("Lookup of %v with method %v (from interface)", c, c.Call.Method)
	fn, err = LookupImpl(info.Prog, c.Call.Method, c.Call.Value)
	if err != nil {
		t.Errorf("cannot find concrete implementation of %v: %v", c, err)
	}
	if expect, got := "*main.t", fn.Signature.Recv().Type().String(); expect != got {
		t.Errorf("Chained #2 lookup wrong:\nExpect:\t%v\nGot:\t%v\n", expect, got)
	}
	t.Logf("%v has type %v", c.Call.Value.Name(), fn.String())
}
