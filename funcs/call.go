package funcs

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"log"

	"github.com/nickng/gospal/store"
	"golang.org/x/tools/go/ssa"
)

// A Call is a function call definition, i.e. function definition from the
// perspective of the caller.
//
// Args are the function arguments at the caller.
type Call struct {
	def        *Definition // Function definition.
	Parameters []store.Key // Aggregated parameter.
	Args       []store.Key // Caller arguments (raw).
	Returns    []store.Key // Caller return values (raw).
}

// MakeCall instantiates function call given its definition and a CallCommon.
func MakeCall(d *Definition, call *ssa.CallCommon, ret ssa.Value) *Call {
	c := Call{def: d}
	// Function call arguments.
	if call != nil {
		c.Args = getArgs(call)
	} else {
		c.Args = getFakeArgs(d.Function)
	}
	if len(c.Args) != d.NParam {
		log.Fatalf("Mismatched argument(%d)/parameter(%d)\n\t%s\n\t%s",
			len(c.Args), d.NParam,
			d.Function.Prog.Fset.Position(call.Pos()).String(),
			d.Function.Prog.Fset.Position(d.Function.Pos()).String())
	}
	if len(d.bindings) != d.NFreeVar {
		log.Printf("Mismatched capture(%d)/binding(%d)\n\t%s\n\t%s",
			len(d.bindings), d.NFreeVar,
			d.Function.Prog.Fset.Position(call.Pos()).String(),
			d.Function.Prog.Fset.Position(d.Function.Pos()).String())
	}
	c.Parameters = make([]store.Key, d.NParam+d.NFreeVar+d.NReturn)
	for i, arg := range c.Args {
		c.Parameters[i] = arg
	}
	for i, binding := range d.bindings {
		c.Parameters[d.NParam+i] = store.Key(binding)
	}
	// Return values are reverse-mapped from body to parameter.
	switch d.NReturn {
	case 0:
	case 1:
		if ret != nil {
			c.Parameters[d.NParam+d.NFreeVar] = ret
		} else {
			c.Parameters[d.NParam+d.NFreeVar] = store.Unused{store.MockKey{
				Description: "Unused_RetVal",
				Typ:         d.Function.Signature.Results().At(0).Type(),
				SrcPos:      d.Function.Pos(),
			}}
		}
	default:
		retvals := make([]store.Key, d.NReturn)
		if ret != nil {
			for _, instr := range *ret.Referrers() {
				switch instr := instr.(type) {
				case *ssa.Extract:
					if instr.Tuple != ret {
						log.Fatal("Return values:", ret.Name(), "is not a tuple")
					}
					c.Parameters[d.NParam+d.NFreeVar+instr.Index] = instr
					retvals[instr.Index] = instr

				case *ssa.DebugRef:
					// ignore
				}
			}
		}
		for i, retval := range retvals {
			if retval == nil {
				c.Parameters[d.NParam+d.NFreeVar+i] = store.Unused{store.MockKey{
					Description: "Unused_RetVal",
					Typ:         d.Function.Signature.Results().At(i).Type(),
					SrcPos:      d.Function.Pos(),
				}}
			}
		}
	}
	return &c
}

func (c *Call) Definition() *Definition {
	return c.def
}

func (c *Call) Param(i int) store.Key   { return c.Parameters[i] }
func (c *Call) Bind(i int) store.Key    { return c.Parameters[c.NParam()+i] }
func (c *Call) Return(i int) store.Key  { return c.Parameters[c.NParam()+c.NBind()+i] }
func (c *Call) Function() *ssa.Function { return c.def.Function }
func (c *Call) NParam() int             { return c.def.NParam }
func (c *Call) NBind() int              { return c.def.NFreeVar }
func (c *Call) NReturn() int            { return c.def.NReturn }

func (c *Call) UniqName() string {
	return c.def.UniqName()
}

func (c *Call) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("funccall(%d): ", len(c.Parameters)))

	if pkg := c.Function().Package(); pkg != nil {
		buf.WriteString(fmt.Sprintf("\"%s\"", pkg.Pkg.Path()))
	}
	buf.WriteString(fmt.Sprintf(".%s ", c.Function().Name()))

	for i := 0; i < c.NParam(); i++ { // Parameters.
		if i > 0 {
			buf.WriteString(", ")
		}
		p := c.Parameters[i]
		buf.WriteString(fmt.Sprintf("%s:%v", p.Name(), p.Type()))
	}
	if c.def.IsVararg {
		buf.WriteRune('…')
	}
	for i := 0; i < c.NBind(); i++ { // Bind variables.
		if i > 0 || c.NParam() > 0 {
			buf.WriteString(", ")
		}
		b := c.Bind(i)
		buf.WriteString(fmt.Sprintf("%s:%v", b.Name(), b.Type()))
	}
	if c.NReturn() > 0 {
		buf.WriteString(" ⇒ {")
		for i := 0; i < c.NReturn(); i++ {
			if i > 0 {
				buf.WriteString(", ")
			}
			retval := c.Return(i)
			buf.WriteString(fmt.Sprintf("%s:%v", retval.Name(), retval.Type()))
		}
		buf.WriteRune('}')
	}
	return buf.String()
}

// getArgs returns arguments in function call.
func getArgs(call *ssa.CallCommon) []store.Key {
	if !call.IsInvoke() { // Call mode.
		switch call.Value.(type) {
		case *ssa.Function, *ssa.MakeClosure:
		case *ssa.Builtin:
		default:
		}
		nArg := len(call.Args)
		args := make([]store.Key, nArg)
		for i, arg := range call.Args {
			args[i] = store.Key(arg)
		}
		return args
	}
	// Invoke mode.
	nArg := len(call.Args) + 1
	args := make([]store.Key, nArg)
	args[0] = call.Value
	for i, arg := range call.Args {
		args[i+1] = arg
	}
	return args
}

// getFakeArgs returns fake arguments in function call.
func getFakeArgs(fn *ssa.Function) []store.Key {
	if sigParam := fn.Signature.Params(); sigParam != nil {
		nArg := sigParam.Len()
		args := make([]store.Key, nArg)
		for i, arg := range fn.Params {
			args[i] = createMock(fn, arg.Type(), "arg")
		}
		return args
	}
	return []store.Key{}
}

// mockValue is a dummy ssa.Value for filling in empty function params/returns.
type mockValue struct {
	parent *ssa.Function // Enclosing function.
	typ    types.Type    // Type of the value.
	desc   string        // Short description.
}

// createMock returns a new mockValue.
func createMock(fn *ssa.Function, t types.Type, s string) store.Key {
	return mockValue{
		parent: fn,
		typ:    t,
		desc:   s,
	}
}

func (m mockValue) Name() string                  { return "_" }
func (m mockValue) Parent() *ssa.Function         { return m.parent }
func (m mockValue) Pos() token.Pos                { return token.NoPos }
func (m mockValue) Referrers() *[]ssa.Instruction { return nil }
func (m mockValue) String() string                { return m.desc }
func (m mockValue) Type() types.Type              { return m.typ }
