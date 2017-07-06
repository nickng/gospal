package funcs

import (
	"bytes"
	"fmt"
	"sync"

	"golang.org/x/tools/go/ssa"
)

var instances struct {
	mu    sync.Mutex
	calls map[*ssa.Function]int
}

// Instantiate materialises a new function call instance.
func Instantiate(call *Call) *Instance {
	if instances.calls == nil {
		instances.calls = make(map[*ssa.Function]int)
	}
	instances.mu.Lock()
	defer instances.mu.Unlock()
	f := call.Function()
	if _, ok := instances.calls[f]; !ok {
		instances.calls[f] = 0
	}
	seq := instances.calls[f]
	instances.calls[f]++
	return &Instance{
		call: call,
		seq:  seq,
	}
}

// An Instance of a function call.
// It is guaranteed unique by the seq field.
type Instance struct {
	call *Call // Function call definition.
	seq  int   // Sequence (instance number).
}

// Call returns the call definition (function definition at caller).
func (i Instance) Call() *Call {
	return i.call
}

// Call returns the function definition (function definition at callee).
func (i Instance) Definition() *Definition {
	return i.call.Definition()
}

// Function returns the function signature/body of the instance.
func (i Instance) Function() *ssa.Function {
	return i.call.Function()
}

func (i Instance) Name() string {
	if i.call == nil {
		return "_emptycall_"
	}
	var buf bytes.Buffer
	if pkg := i.call.Function().Package(); pkg != nil {
		// Package path is a free-form string, so quote it.
		buf.WriteString(fmt.Sprintf("%s", pkg.Pkg.Path()))
	}
	buf.WriteString(fmt.Sprintf(".%s", i.call.Function().Name()))
	return buf.String()
}

func (i Instance) UniqName() string {
	if i.call == nil {
		return "_emptycall_"
	}
	var buf bytes.Buffer
	if pkg := i.call.Function().Package(); pkg != nil {
		// Package path is a free-form string, so quote it.
		buf.WriteString(fmt.Sprintf("\"%s\"", pkg.Pkg.Path()))
	}
	buf.WriteString(fmt.Sprintf(".%s%d", i.call.Function().Name(), i.seq))
	return buf.String()
}
