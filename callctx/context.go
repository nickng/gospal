// Package callctx defines a function call Context for static analysis.
//
// A context is a map between local variables of a function and their instances.
// An element in the context is abstracted as a key-value store using the Key
// type and the Value type.
// The primary use for a context is to propagate local variables to other scopes
// through different forms of function calls using a substitution operation
// called Switch. A context Switch transforms context from the perspective of a
// caller to that of a callee.
//
// This package is inspired by the builtin context package but for
// context-sensitive variable propagation.
//
package callctx

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/store"
	"github.com/nickng/gospal/store/structs"
)

// A Context is map between local variables and their instances.
// Get and Put are the main functions to access the contents of the map.
//
type Context interface {
	Get(store.Key) store.Value
	Put(store.Key, store.Value)
	getStorage() *store.Store
}

// An emptyCtx is a context that contains no variables.
type emptyCtx struct {
	s *store.Store
}

func (c *emptyCtx) Get(store.Key) store.Value  { return nil }
func (c *emptyCtx) Put(store.Key, store.Value) {}
func (c *emptyCtx) getStorage() *store.Store   { return c.s }

var toplevel = &emptyCtx{s: store.New()}

// Toplevel returns an empty context.
//
// It is used for representing a top-level context at entry points of analysis.
func Toplevel() Context {
	return toplevel
}

// Updater is an interface for a context that has the ability to modify the
// underlying storage which the instances point to.
//
type Updater interface {
	PutObj(k store.Key, v ssa.Value)                 // Add Value.
	PutUniq(k store.Key, v store.ValueWrapper) error // Add pre-initialised Value.
}

// A calleeCtx is a context created by a Call.
//
// A callee shares backing storage with caller.
type calleeCtx struct {
	*store.Store
	parent Context
	callee *funcs.Instance
}

func newCalleeCtx(parent Context) calleeCtx {
	ctx := calleeCtx{
		Store:  store.Extend(parent.getStorage()),
		parent: parent,
	}
	return ctx
}

func (c *calleeCtx) getStorage() *store.Store { return c.Store }

// Call returns the function call instance.
func (c *calleeCtx) Call() *funcs.Instance {
	return c.callee
}

func (c *calleeCtx) CallerCtx() Context {
	return c.parent
}

func (c *calleeCtx) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Context: %s\n", c.callee.UniqName()))
	buf.WriteString(c.Store.String())
	return buf.String()
}

// A Callee is a context created by a call, Call() returns the call instance.
type Callee interface {
	CallerCtx() Context
	Call() *funcs.Instance
}

// Switch performs a context switch from parent (caller) to a new callee
// context.
// The switch matches up caller arguments in parent in the call ↔ translated in
// the callee as call parameters.
func Switch(parent Context, call *funcs.Instance) Context {
	c := newCalleeCtx(parent)
	c.callee = call

	for i, arg := range call.Call().Parameters {
		argValue := parent.Get(arg)
		param := call.Definition().Parameters[i]
		if argStruct, ok := argValue.(*structs.Struct); ok {
			paramStruct := structs.New(call, argStruct.Value)
			c.Put(param, paramStruct)

			argFields := argStruct.Expand()
			paramFields := paramStruct.Expand() // All empty.

			for i, argField := range argFields {
				switch sf := argField.(type) {
				case structs.SField:
					paramField := paramFields[i].(structs.SField)
					paramField.Struct.Fields[paramField.Index] = paramField
					if sf.Key != nil {
						argFieldVal := parent.Get(sf.Key)
						c.Put(paramField, argFieldVal)
					}
				case *structs.Struct:
					// Skip. The fields would be handled above after Expand()
				}
			}
		} else {
			if param != nil {
				c.Put(param, argValue)
			}
		}
	}
	return &c
}

func Deref(ctx Context, ptr, val store.Key) (store.Value, error) {
	if t, ok := ptr.Type().(*types.Pointer); ok && types.Identical(t.Elem(), val.Type()) {
		inst := ctx.Get(ptr)
		ctx.Put(val, inst)
		return inst, nil
	}
	return nil, errors.New("incompatible type")
}
