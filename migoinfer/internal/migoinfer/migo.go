package migoinfer

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"

	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/store"
	"github.com/nickng/gospal/store/chans"
	"github.com/nickng/gospal/store/structs"
	"github.com/nickng/migo"
)

// MiGo specific helpers.

// Exported is a holder of visible local names in a function.
type Exported struct {
	names []store.Key
}

// Export puts a local variable k in the set of exported names.
// Only exported names can appear in a MiGo function.
//
// Precondition: e only has unique elements.
func (e *Exported) Export(k store.Key) {
	for _, name := range e.names {
		if name.Name() == k.Name() {
			return
		}
	}
	e.names = append(e.names, k)
}

func (e *Exported) Unexport(k store.Key) {
	for i := 0; i < len(e.names); i++ {
		if e.names[i].Name() == k.Name() {
			e.names = append(e.names[:i], e.names[i+1:]...)
			return
		}
	}
}

// FindExported returns name that points the same value v but using an exported
// name.
func (e *Exported) FindExported(ctx callctx.Context, v store.Value) store.Key {
	if e.names == nil {
		return Unexported{Key: store.MockKey{Description: "Unexported value"}, Value: v}
	}
	for _, name := range e.names {
		if ctx.Get(name) == v {
			return name
		}
	}
	return Unexported{Key: store.MockKey{Description: "Unexported value"}, Value: v}
}

func (e *Exported) String() string {
	var buf bytes.Buffer
	buf.WriteString("┌─ Exported\n")
	for _, n := range e.names {
		buf.WriteString(fmt.Sprintf("│ %s\n", n.Name()))
	}
	buf.WriteString("└─")
	return buf.String()
}

type Unexported struct {
	store.Key
	Value store.Value
}

// migoCall returns a 'call' in MiGo using exported values.
func migoCall(fn string, blk *ssa.BasicBlock, exported *Exported) migo.Statement {
	var params []*migo.Parameter
	for _, name := range exported.names {
		params = append(params, &migo.Parameter{Caller: name, Callee: name})
	}
	// Remove φ names that does not belong in the target block.
	// call f([x ↦ a][y ↦ c]) becomes call f([x ↦ a])
	// def f(x):
	//   y = φ[a,b]
	for _, instr := range blk.Instrs {
		switch instr := instr.(type) {
		case *ssa.Phi:
			removed := 0
			for i := range params {
				if instr.Name() == params[i-removed].Caller.Name() {
					params = append(params[:i-removed], params[i-removed+1:]...)
					removed++
				}
			}
		}
	}
	if blk.Index == 0 {
		return &migo.CallStatement{Name: fn, Params: params}
	}
	return &migo.CallStatement{Name: fmt.Sprintf("%s#%d", fn, blk.Index), Params: params}
}

// migoNewChan returns a 'newchan' in MiGo.
func migoNewChan(v *Logger, name migo.NamedVar, ch *chans.Chan) migo.Statement {
	v.Debugf("%s migo newchan name=%v value=%v", v.Module(), name, ch)
	return &migo.NewChanStatement{Name: name, Chan: ch.UniqName(), Size: ch.Size()}
}

// migoNilChan returns a nil 'newchan' in MiGo.
// The nilchan uses the given key k to generate a let statement.
func migoNilChan(v *Instruction, k store.Key) migo.Statement {
	v.Debugf("%s migo nilchan name=%v", v.Module(), k.Name())
	return &migo.NewChanStatement{Name: k, Chan: "nilchan", Size: 0}
}

// freshNilchan is a datastructure to represent 'fresh' unnamed nil channel.
type freshNilChan struct {
	count int        // Fresh nilchan index.
	typ   types.Type // Type of given nil chan.
}

func newFreshNilChan(t types.Type) freshNilChan {
	defer func() { nextNilChan++ }()
	return freshNilChan{count: nextNilChan, typ: t}
}

// nextNilChan keeps track of current count of unnamed fresh nilchan.
var nextNilChan int

func (n freshNilChan) Name() string     { return fmt.Sprintf("nil%d", n.count) }
func (n freshNilChan) Pos() token.Pos   { return token.NoPos }
func (n freshNilChan) Type() types.Type { return n.typ }
func (n freshNilChan) String() string {
	return fmt.Sprintf("[nilchan%d:%s]", n.count, n.Type().String())
}

// timeChan returns true if given ch is created by time.*.
func timeChan(ch store.Key) bool {
	isTimeFunc := func(c *ssa.Call) bool {
		fn := c.Call.StaticCallee()
		return fn != nil && fn.Pkg != nil && fn.Pkg.Pkg.Path() == "time"
	}
	switch instr := ch.(type) {
	case *ssa.Call:
		return isTimeFunc(instr)
	case *ssa.UnOp:
		if instr.Op == token.MUL {
			if fa, ok := instr.X.(*ssa.FieldAddr); ok {
				if c, ok := fa.X.(*ssa.Call); ok {
					return isTimeFunc(c)
				}
			}
		}
	}
	return false
}

// migoRecv returns a Receive Statement in MiGo.
func migoRecv(v *Instruction, local store.Key, ch store.Value) migo.Statement {
	if timeChan(local) {
		v.Debugf("%s migo recv name=%v (time chan, replace with τ)", v.Module(), local)
		return &migo.TauStatement{}
	}

	v.Debugf("%s migo recv name=%v, value=%s", v.Module(), local, ch.UniqName())
	if c, ok := local.(*ssa.Const); ok {
		if c.IsNil() {
			nc := newFreshNilChan(local.Type())
			v.MiGo.AddStmts(migoNilChan(v, nc))
			return &migo.RecvStatement{Chan: nc.Name()}
		}
	}
	if u, ok := local.(*ssa.UnOp); ok && u.Op == token.MUL { // Deref
		// Use deref'd versions: u.X ⇒ local, v.Get(u.X) ⇒ ch instead.
		local, ch = u.X, v.Get(u.X)
	}
	switch exported := v.FindExported(v.Context, ch).(type) {
	case Unexported:
		v.Warnf("%s Channel %s/%s unavail. in current scope (unexported)\n\t%s",
			v.Module(), local.Name(), ch.UniqName(), v.Env.getPos(local))
		if _, isField := local.(structs.SField); !isField { // If not defined as a struct-field.
			v.MiGo.AddStmts(migoNilChan(v, local))
		}
		return &migo.RecvStatement{Chan: local.Name()}
	default:
		// Channel exists and exported: this is the name we want to receive.
		v.Debugf("%s Receive %s⇔%s ↦ %s\t%s",
			v.Module(), local.Name(), exported.Name(), ch.UniqName(), local.Type())
		return &migo.RecvStatement{Chan: exported.Name()}
	}
}

// migoSend returns a Send Statement in MiGo.
func migoSend(v *Instruction, local store.Key, ch store.Value) migo.Statement {
	v.Debugf("%s migo send name=%v, value=%s", v.Module(), local, ch.UniqName())
	if c, ok := local.(*ssa.Const); ok {
		if c.IsNil() {
			nc := newFreshNilChan(local.Type())
			v.MiGo.AddStmts(migoNilChan(v, nc))
			return &migo.SendStatement{Chan: nc.Name()}
		}
	}
	if u, ok := local.(*ssa.UnOp); ok && u.Op == token.MUL { // Deref
		// Use deref'd versions: u.X ⇒ local, v.Get(u.X) ⇒ ch instead.
		local, ch = u.X, v.Get(u.X)
	}
	switch exported := v.FindExported(v.Context, ch).(type) {
	case Unexported:
		v.Warnf("%s Channel %s/%s unavail. in current scope (unexported)\n\t%s",
			v.Module(), local.Name(), ch.UniqName(), v.Env.getPos(local))
		if _, isField := local.(structs.SField); !isField { // If not defined as a struct-field.
			v.MiGo.AddStmts(migoNilChan(v, local))
		}
		return &migo.SendStatement{Chan: local.Name()}
	default:
		// Channel exists and exported: this is the name we want to send.
		v.Debugf("%s Send %s⇔%s ↦ %s\t%s",
			v.Module(), local.Name(), exported.Name(), ch.UniqName(), local.Type())
		return &migo.SendStatement{Chan: exported.Name()}
	}
}

// isDefinedMiGoName checks that given name is defined.
//
// The primary use of this function is for detecting nilchan within MiGo def.
// Defined here means either name is in def parameter (always use the parameter)
// or name is defined by MiGo let/newchan but not used.
func isDefinedMiGoName(v *Instruction, name store.Key) bool {
	for _, param := range v.MiGo.Params {
		if param.Callee.Name() == name.Name() {
			v.Debugf("%s %s was a MiGo parameter in def %s",
				v.Module(), name.Name(), v.MiGo.SimpleName())
			return true
		}
	}
	defined := false
	for _, stmt := range v.MiGo.Stmts {
		switch stmt := stmt.(type) {
		case *migo.NewChanStatement:
			if stmt.Name.Name() == name.Name() {
				v.Debugf("%s %s was a MiGo name defined in def %s",
					v.Module(), name.Name(), v.MiGo.SimpleName())
				defined = true
			}
		case *migo.SpawnStatement:
			// If channel is used, reset to undefined (needs redefining).
			for _, param := range stmt.Params {
				if param.Caller.Name() == name.Name() {
					defined = false
				}
			}
		case *migo.CallStatement:
			// If channel is used, reset to undefined (needs redefining).
			for _, param := range stmt.Params {
				if param.Caller.Name() == name.Name() {
					defined = false
				}
			}
		}
	}
	return defined
}

// paramsToMigoParam converts call parameters into MiGo parameters if they are
// channel types.
func paramsToMigoParam(v *Instruction, fn *Function, call *funcs.Call) []*migo.Parameter {
	// Converts an argument and a function parameter pair to migo Parameter.
	convertToMigoParam := func(arg, param store.Key) *migo.Parameter {
		switch ch := v.Get(arg).(type) {
		case store.MockValue:
			if _, isPhi := arg.(*ssa.Phi); isPhi {
				v.Warnf("%s Undefined argument %s is Phi ⇔ %v",
					v.Module(), arg,
					&migo.Parameter{Caller: arg, Callee: param})
			} else {
				field, isField := arg.(structs.SField)
				if isField && field.Key != nil {
					// Is field and is defined.
				} else {
					v.Warnf("%s Argument %v undefined → nil chan.\n\t%s",
						v.Module(), arg, v.Env.getPos(arg))
					if isField && !isDefinedMiGoName(v, field) {
						v.MiGo.AddStmts(migoNilChan(v, field))
					} else if !isDefinedMiGoName(v, arg) {
						v.MiGo.AddStmts(migoNilChan(v, arg))
					}
				}
			}
		case *chans.Chan:
			if exported := v.FindExported(v.Context, ch); exported != nil {
				arg = exported
			}
		}
		return &migo.Parameter{Caller: arg, Callee: param}
	}

	var migoParams []*migo.Parameter
	for i, arg := range call.Parameters[:call.NParam()+call.NBind()] {
		arg := underlying(arg)
		param := underlying(call.Definition().Param(i))
		if isStruct(arg) && isStruct(param) {
			argStruct := v.Get(arg)
			paramStruct := fn.Get(param)
			if mock, ok := argStruct.(store.MockValue); ok {
				v.Debugf("%s %s is a nil struct (arg) (type:%s)",
					v.Module(), arg.Name(), arg.Type().String())
				argStruct = structs.New(mock, arg)
			} else if _, ok := argStruct.(*structs.Struct); !ok {
				argStruct = structs.New(v.Callee, arg.(ssa.Value))
			}
			if mock, ok := paramStruct.(store.MockValue); ok {
				v.Debugf("%s %s is a nil struct (param) (type:%s)",
					v.Module(), param.Name(), param.Type().String())
				paramStruct = structs.New(mock, param.(ssa.Value))
			} else if _, ok := paramStruct.(*structs.Struct); !ok {
				paramStruct = structs.New(v.Callee, arg.(ssa.Value))
			}
			argFields := argStruct.(*structs.Struct).Expand()
			paramFields := paramStruct.(*structs.Struct).Expand()
			for i := 0; i < len(argFields); i++ {
				switch argField := argFields[i].(type) {
				case structs.SField:
					if isChan(argField) {
						migoParams = append(migoParams, convertToMigoParam(argField, paramFields[i]))
					}
				case *structs.Struct:
					// Ignore.
				}
			}
		} else if isStruct(arg) && !isStruct(param) && types.IsInterface(param.Type()) {
			// Skips struct arg/param pair-up.
			v.Debugf("%s Function argument is struct (type:%s), parameter is not (type:%s), likely a wildcard interface{}",
				v.Module(), arg.Type().String(), param.Type().String())
		}
		if isChan(arg) {
			migoParams = append(migoParams, convertToMigoParam(arg, call.Definition().Param(i)))
		}
	}
	// Convert return value.
	for i, param := range call.Parameters[call.NParam()+call.NBind():] {
		if isChan(param) {
			migoParam := &migo.Parameter{Caller: param, Callee: call.Definition().Return(i)}
			if exported := fn.FindExported(fn.Context, fn.Get(call.Definition().Return(i))); exported != nil {
				migoParam.Callee = exported
				for j := range migoParams {
					if migoParams[j].Callee.Name() == exported.Name() {
						migoParam.Caller = migoParams[j].Caller
					}
				}
			}
			migoParams = append(migoParams, migoParam)
		}
	}
	return migoParams
}

// underlying returns the underlying value after type assertion/interface.
func underlying(v store.Key) store.Key {
	switch v := v.(type) {
	case *ssa.MakeInterface:
		return underlying(v.X)
	case *ssa.TypeAssert:
		return underlying(v.X)
	}
	return v
}
