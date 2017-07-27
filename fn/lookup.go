package fn

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/ssa"
)

var (
	ErrNilMeth      = errors.New("interface method is nil")
	ErrNilImpl      = errors.New("interface implementation is nil")
	ErrAbstractMeth = errors.New("interface method is abstract")
)

// MethTypeError is the error when interface Iface with method Meth is
// implemented as Impl with a wrong type.
// See also go/types.MissingMethod.
type MethTypeError struct {
	Iface *types.Interface // Interface to implement.
	Meth  *types.Func      // Method of the interface.
	Impl  *types.Func      // Implemented method.
}

func (e MethTypeError) Error() string {
	return fmt.Sprintf("type error: %v (interface %v has method %s of type %s)",
		e.Meth, e.Iface, e.Impl.Name(), e.Impl.Type())
}

// MethNotFoundError is the error when an interface Iface is implemented without
// method Meth.
type MethNotFoundError struct {
	Iface *types.Interface // Interface to look in.
	Meth  *types.Func      // Expected method signature.
}

func (e MethNotFoundError) Error() string {
	return fmt.Sprintf("missing method: %v (interface %v does not have method %s)",
		e.Meth, e.Iface, e.Meth.Name())
}

// DoesNotImplError is the error when a supplied implementation Impl does not
// implement an interface Iface.
type DoesNotImplError struct {
	Iface *types.Interface
	Impl  ssa.Value
}

type UnknownInvokeError struct {
	Iface *types.Interface
	Impl  ssa.Value
}

func (e UnknownInvokeError) Error() string {
	return fmt.Sprintf("Unknown implementation of interface %v: %v (type: %v)",
		e.Iface, e.Impl, e.Impl.Type())
}

// LookupImpl finds concrete implementation Function of a given
// interface/abstract type.
func LookupImpl(prog *ssa.Program, meth *types.Func, impl ssa.Value) (*ssa.Function, error) {
	if meth == nil {
		return nil, ErrNilMeth
	}
	// isIface true:  Makes sure iface is a subtype of impl (static check).
	// isIface false: Normal check (non-dynamic check).
	iface, isIface := impl.Type().Underlying().(*types.Interface)

	// Make sure impl has meth.
	missing, wrongType := types.MissingMethod(impl.Type().Underlying(), iface, isIface)
	if missing != nil {
		if wrongType {
			return nil, MethTypeError{Iface: iface, Meth: meth, Impl: missing}
		}
		return nil, MethNotFoundError{Meth: missing}
	}
	switch t := concreteImpl(impl).(type) {
	case *ssa.Alloc:
		if fn := prog.LookupMethod(t.Type(), meth.Pkg(), meth.Name()); fn != nil {
			return fn, nil
		}
		return nil, ErrAbstractMeth
	case *ssa.Extract:
		// Implementation is a tuple.
		if fn := prog.LookupMethod(t.Type(), meth.Pkg(), meth.Name()); fn != nil {
			return fn, nil
		}
		return nil, ErrAbstractMeth
	case *ssa.Parameter:
		if fn := prog.LookupMethod(t.Type(), meth.Pkg(), meth.Name()); fn != nil {
			return fn, nil
		}
		return nil, ErrAbstractMeth
	case *ssa.Phi:
		// Merging of implementation (e.g. by reflection)
		// The edges are not important as long as they are type checked
		// and the Phi value's type is used.
		if fn := prog.LookupMethod(t.Type(), meth.Pkg(), meth.Name()); fn != nil {
			return fn, nil
		}
		return nil, ErrAbstractMeth
	default:
		log.Printf("LookupImpl: Unknown invoke implementation (type %T): got %+v.%v\n\t%s",
			t, t, meth,
			prog.Fset.Position(impl.Pos()))
		return nil, UnknownInvokeError{Iface: iface, Impl: impl}
	}
}

// concreteImpl finds the SSA value with the most concrete type.
func concreteImpl(v ssa.Value) ssa.Value {
	switch instr := v.(type) {
	case *ssa.Call:
		if instr.Call.IsInvoke() {
			return concreteImpl(instr.Call.Value) // use return value.
		}
		if fn := instr.Call.StaticCallee(); fn != nil && len(fn.Blocks) > 0 {
			return concreteImpl(fnBodyRetval(fn)) // use return value from func body.
		}
	case *ssa.MakeInterface:
		return concreteImpl(instr.X) // revert interface to original struct.
	case *ssa.TypeAssert:
		return concreteImpl(instr.X) // revert assert to original.
	case *ssa.UnOp:
		if instr.Op == token.MUL {
			switch instr.Type().Underlying().(type) {
			case *types.Struct:
				return concreteImpl(instr.X)
			case *types.Interface: // Interface is always a pointer, so don't need to deref.
				return instr
			}
		}
	}
	return v
}

// fnBodyRetval returns the first return value of the function.
// This does not have to be accurate as we only need to know the type.
func fnBodyRetval(fn *ssa.Function) (retval ssa.Value) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr := instr.(type) {
			case *ssa.Return:
				retval = instr.Results[0]
			}
		}
	}
	return
}
