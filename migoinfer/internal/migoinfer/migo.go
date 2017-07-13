package migoinfer

import (
	"bytes"
	"fmt"
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
func migoNewChan(name migo.NamedVar, ch *chans.Chan) migo.Statement {
	return &migo.NewChanStatement{Name: name, Chan: ch.UniqName(), Size: ch.Size()}
}

// migoRecv returns a Receive Statement in MiGo.
func migoRecv(v *Instruction, local ssa.Value, ch store.Value) migo.Statement {
	if _, ok := ch.(store.MockValue); !ok {
		switch param := v.FindExported(v.Context, ch).(type) {
		case Unexported:
			v.Logger.Warnf("%s Channel %s/%s not exported in current scope\n\t%s",
				v.Logger.Module(), local.Name(), ch.UniqName(), v.Env.getPos(local))
			return (&migo.RecvStatement{Chan: param.Name()})
		default:
			v.Logger.Debugf("%s Receive %s==%s ↦ %s\t%s",
				v.Logger.Module(), local.Name(), param.Name(), ch.UniqName(), local.Type())
			return (&migo.RecvStatement{Chan: param.Name()})
		}
	}
	v.Logger.Warnf("%s Receive unknown-channel %s\n\t%s",
		v.Logger.Module(), ch.UniqName(), v.Env.getPos(local))
	return (&migo.RecvStatement{Chan: local.Name()})
}

// migoSend returns a Send Statement in MiGo.
func migoSend(v *Instruction, local ssa.Value, ch store.Value) migo.Statement {
	if _, ok := ch.(store.MockValue); !ok {
		switch param := v.FindExported(v.Context, ch).(type) {
		case Unexported:
			v.Logger.Warnf("%s Channel %s/%s not exported in current scope\n\t%s",
				v.Logger.Module(), local.Name(), ch.UniqName(), v.Env.getPos(local))
			return &migo.SendStatement{Chan: param.Name()}
		default:
			v.Logger.Debugf("%s Send %s==%s ↦ %s\t%s",
				v.Logger.Module(), local.Name(), param.Name(), ch.UniqName(), local.Type())
			return &migo.SendStatement{Chan: param.Name()}
		}
	}
	v.Logger.Warnf("%s Send unknown-channel %s\n\t%s",
		v.Logger.Module(), ch.UniqName(), v.Env.getPos(local))
	return &migo.SendStatement{Chan: local.Name()}
}

// paramsToMigoParam converts call parameters into MiGo parameters if they are
// channel types.
func paramsToMigoParam(v *Instruction, fn *Function, call *funcs.Call) []*migo.Parameter {
	// Converts an argument and a function parameter pair to migo Parameter.
	convertToMigoParam := func(arg, param store.Key) *migo.Parameter {
		callArg := arg
		switch ch := v.Get(callArg).(type) {
		case store.MockValue:
			if _, isPhi := callArg.(*ssa.Phi); isPhi {
				v.Logger.Warnf("%s Undefined argument %s is Phi == %v",
					v.Logger.Module(), callArg,
					&migo.Parameter{Caller: callArg, Callee: param})
			} else {
				v.Logger.Warnf("%s Argument %v undefined → nil chan.\n\t%s",
					v.Logger.Module(), callArg, v.Env.getPos(callArg))
			}
		case *chans.Chan:
			if exported := v.FindExported(v.Context, ch); exported != nil {
				callArg = exported
			}
		}
		return &migo.Parameter{Caller: callArg, Callee: param}
	}

	var migoParams []*migo.Parameter
	for i, arg := range call.Parameters {
		arg := underlying(arg)
		param := underlying(call.Definition().Param(i))
		if isStruct(arg) && isStruct(param) {
			argStruct := v.Get(arg)
			paramStruct := fn.Get(param)
			if mock, ok := argStruct.(store.MockValue); ok {
				v.Logger.Debugf("%s %s is a nil struct (arg) (type:%s)",
					v.Logger.Module(), arg.Name(), arg.Type().String())
				argStruct = structs.New(mock, arg.(ssa.Value))
			} else if _, ok := argStruct.(*structs.Struct); !ok {
				argStruct = structs.New(v.Callee, arg.(ssa.Value))
			}
			if mock, ok := paramStruct.(store.MockValue); ok {
				v.Logger.Debugf("%s %s is a nil struct (param) (type:%s)",
					v.Logger.Module(), param.Name(), param.Type().String())
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
			v.Logger.Debugf("%s Function argument is struct (type:%s), parameter is not (type:%s), likely a wildcard interface{}",
				v.Logger.Module(), arg.Type().String(), param.Type().String())
		}
		if isChan(arg) {
			migoParams = append(migoParams, convertToMigoParam(arg, call.Definition().Param(i)))
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
