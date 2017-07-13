package migoinfer

import (
	"go/token"
	"go/types"

	"github.com/fatih/color"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/fn"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/store"
	"github.com/nickng/gospal/store/chans"
	"github.com/nickng/gospal/store/structs"
	"github.com/nickng/migo"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/ssa"
)

// Instruction is a visitor for related instructions within a block.
type Instruction struct {
	Callee          *funcs.Instance // Instance of this function.
	callctx.Context                 // Function context.
	Env             *Environment    // Program environment.

	MiGo      *migo.Function // MiGo function definition of current block.
	*Exported                // Local variables.
	*Logger
}

func NewInstruction(callee *funcs.Instance, ctx callctx.Context, env *Environment, migoFn *migo.Function) *Instruction {
	i := Instruction{
		Callee:  callee,
		Context: ctx,
		Env:     env,
		MiGo:    migoFn,
	}
	return &i
}

func (v *Instruction) VisitInstr(instr ssa.Instruction) {
	switch instr := instr.(type) {
	case *ssa.Alloc:
		v.Logger.Debugf("%s Alloc: %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitAlloc(instr)

	case *ssa.BinOp:
		v.VisitBinOp(instr)

	case *ssa.Call:
		v.Logger.Debugf("%s Call: %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitCall(instr)

	case *ssa.ChangeInterface:
		v.VisitChangeInterface(instr)

	case *ssa.ChangeType:
		v.Logger.Debugf("%s ChangeType: %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitChangeType(instr)

	case *ssa.Convert:
		v.VisitConvert(instr)

	case *ssa.DebugRef:
		v.VisitDebugRef(instr)

	case *ssa.Defer:
		v.VisitDefer(instr)

	case *ssa.Extract:
		v.VisitExtract(instr)

	case *ssa.Field:
		v.VisitField(instr)

	case *ssa.FieldAddr:
		v.Logger.Debugf("%s FieldAddr: %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitFieldAddr(instr)

	case *ssa.Go:
		v.Logger.Debugf("%s Go: %s\n\t%s",
			v.Logger.Module(), instr, v.Env.getPos(instr))
		v.VisitGo(instr)

	case *ssa.If:
		v.VisitIf(instr)

	case *ssa.Index:
		v.VisitIndex(instr)

	case *ssa.IndexAddr:
		v.VisitIndexAddr(instr)

	case *ssa.Jump:
		v.VisitJump(instr)

	case *ssa.Lookup:
		v.VisitLookup(instr)

	case *ssa.MakeChan:
		v.Logger.Debugf("%s MakeChan %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitMakeChan(instr)

	case *ssa.MakeClosure:
		v.Logger.Debugf("%s MakeClosure %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitMakeClosure(instr)

	case *ssa.MakeInterface:
		v.Logger.Debugf("%s MakeInterface %s = %s\n\t%s",
			v.Logger.Module(), instr.Name(), instr, v.Env.getPos(instr))
		v.VisitMakeInterface(instr)

	case *ssa.MakeMap:
		v.VisitMakeMap(instr)

	case *ssa.MakeSlice:
		v.VisitMakeSlice(instr)

	case *ssa.MapUpdate:
		v.VisitMapUpdate(instr)

	case *ssa.Next:
		v.VisitNext(instr)

	case *ssa.Panic:
		v.VisitPanic(instr)

	case *ssa.Phi:
		v.VisitPhi(instr)

	case *ssa.Range:
		v.VisitRange(instr)

	case *ssa.Return:
		v.VisitReturn(instr)

	case *ssa.RunDefers:
		v.VisitRunDefers(instr)

	case *ssa.Select:
		v.VisitSelect(instr)

	case *ssa.Send:
		v.Logger.Debugf("%s Send: %s\n\t%s",
			v.Logger.Module(), instr, v.Env.getPos(instr))
		v.VisitSend(instr)

	case *ssa.Slice:
		v.VisitSlice(instr)

	case *ssa.Store:
		v.VisitStore(instr)

	case *ssa.TypeAssert:
		v.VisitTypeAssert(instr)

	case *ssa.UnOp:
		v.VisitUnOp(instr)

	default:
		v.Logger.Fatalf("%s Unhandled instruction %q (%T)\n\t%s",
			v.Logger.Module(), instr, instr, v.Env.getPos(instr))
	}
}

func (v *Instruction) VisitAlloc(instr *ssa.Alloc) {
	t := instr.Type().(*types.Pointer).Elem()
	switch t := t.Underlying().(type) {
	case *types.Struct:
		v.Logger.Debugf("%s Allocate struct: %T", v.Logger.Module(), t)
		if updater, ok := v.Context.(callctx.Updater); ok {
			updater.PutUniq(instr, structs.New(v.Callee, instr))
		}
	default:
		v.Logger.Debugf("%s Alloc %s = type %s (delay write)",
			v.Logger.Module(), instr.Name(), t.String())
	}
}

func (v *Instruction) VisitBinOp(instr *ssa.BinOp) {
}

func (v *Instruction) VisitCall(instr *ssa.Call) {
	def := v.createDefinition(instr.Common())
	if def == nil {
		return
	}
	if _, ok := v.Env.VisitedFunc[instr.Common()]; ok {
		return
	}
	v.Env.VisitedFunc[instr.Common()] = true
	v.doCall(instr, def)
}

func (v *Instruction) VisitChangeInterface(instr *ssa.ChangeInterface) {
}

func (v *Instruction) VisitChangeType(instr *ssa.ChangeType) {
	if _, ok := instr.X.Type().(*types.Chan); ok {
		v.Put(instr, v.Get(instr.X))
	}
}

func (v *Instruction) VisitConvert(instr *ssa.Convert) {
}

func (v *Instruction) VisitDebugRef(instr *ssa.DebugRef) {
}

func (v *Instruction) VisitDefer(instr *ssa.Defer) {
}

func (v *Instruction) VisitExtract(instr *ssa.Extract) {
}

func (v *Instruction) VisitField(instr *ssa.Field) {
}

func (v *Instruction) VisitFieldAddr(instr *ssa.FieldAddr) {
	switch struc := v.Get(instr.X).(type) {
	case *structs.Struct:
		if field := struc.Fields[instr.Field]; field != nil {
			if fieldVal := v.Get(field); fieldVal != nil {
				v.Logger.Debugf("%s Field %s exists, replacing with %s",
					v.Logger.Module(), field, instr.Name())
				v.Put(instr, fieldVal)
				// If field is a FieldParam, replace field key and export.
				if _, ok := field.(structs.FieldParam); ok {
					struc.Fields[instr.Field] = instr
					if isChan(instr) {
						v.Export(instr)
					}
				}
			}
		} else {
			// Put object in storage.
			if updater, ok := v.Context.(callctx.Updater); ok {
				updater.PutObj(instr, instr.X)
			}
			struc.Fields[instr.Field] = instr
		}
	case *store.MockValue:
		v.Logger.Debugf("%s struct undefined\n\t%s",
			v.Logger.Module(), v.Env.getPos(instr))
	default:
		v.Logger.Warnf("%s FieldAddr: %v is not a struct\t%s\n\t%s",
			v.Logger.Module(), instr.X, instr.X.Type().Underlying(), v.Env.getPos(instr))
	}
}

func (v *Instruction) VisitGo(instr *ssa.Go) {
	def := v.createDefinition(instr.Common())
	if def == nil {
		return
	}
	if _, ok := v.Env.VisitedFunc[instr.Common()]; ok {
		return
	}
	v.Env.VisitedFunc[instr.Common()] = true
	v.doGo(instr, def)
}

func (v *Instruction) VisitIf(instr *ssa.If) {
}

func (v *Instruction) VisitIndex(instr *ssa.Index) {
}

func (v *Instruction) VisitIndexAddr(instr *ssa.IndexAddr) {
}

func (v *Instruction) VisitJump(instr *ssa.Jump) {
}

func (v *Instruction) VisitLookup(instr *ssa.Lookup) {
}

func (v *Instruction) VisitMakeChan(instr *ssa.MakeChan) {
	newch := v.newChan(instr)
	isReturnValue := v.Callee.Definition().IsReturn(instr)
	var isParameter bool
	str, field, isField := getStruct(instr)
	if !isField { // Could be that the field is stored by *t0 = make(chan)
		for _, ref := range *instr.Referrers() {
			switch ref := ref.(type) {
			case *ssa.Store:
				if ref.Val == instr {
					str, field, isField = getStruct(ref.Addr)
				}
			}
		}
	}
	if isField {
		for _, param := range v.Callee.Definition().Parameters[:v.Callee.Definition().NParam+v.Callee.Definition().NFreeVar] {
			if param == str {
				isParameter = true
			}
		}
		if s, ok := v.Get(str).(*structs.Struct); ok {
			s.Fields[field] = instr
		}
	}
	if isReturnValue || isParameter {
		v.Logger.Debugf("%s %s = MakeChan skipped\n\treturn value? %t\n\tparameter? %t",
			v.Logger.Module(),
			instr.Name(), isReturnValue, isParameter)
		v.MiGo.AddStmts(&migo.TauStatement{})
		return
	}
	v.Put(instr, newch)
	v.Export(instr)
	v.MiGo.AddStmts(migoNewChan(instr, newch))
}

func (v *Instruction) VisitMakeClosure(instr *ssa.MakeClosure) {
	def := funcs.MakeClosureDefinition(instr.Fn.(*ssa.Function), instr.Bindings)
	v.Put(instr, def)    // For calling the closure.
	v.Put(instr.Fn, def) // For reusing the closure.
	f := v.Get(instr)
	v.Logger.Debugf("%s ↳ %s", v.Logger.Module(), f.(*funcs.Definition).String())
}

func (v *Instruction) VisitMakeInterface(instr *ssa.MakeInterface) {
	iface := v.Get(instr.X)
	v.Logger.Debugf("%s iface → %v", v.Logger.Module(), iface)
	v.Put(instr, iface)
}

func (v *Instruction) VisitMakeMap(instr *ssa.MakeMap) {
}

func (v *Instruction) VisitMakeSlice(instr *ssa.MakeSlice) {
}

func (v *Instruction) VisitMapUpdate(instr *ssa.MapUpdate) {
}

func (v *Instruction) VisitNext(instr *ssa.Next) {
}

func (v *Instruction) VisitPanic(instr *ssa.Panic) {
}

func (v *Instruction) VisitPhi(instr *ssa.Phi) {
}

func (v *Instruction) VisitRange(instr *ssa.Range) {
}

func (v *Instruction) VisitReturn(instr *ssa.Return) {
}

func (v *Instruction) VisitRunDefers(instr *ssa.RunDefers) {
}

func (v *Instruction) VisitSelect(instr *ssa.Select) {
	v.MiGo.AddStmts(v.getSelectCases(instr))
}

func (v *Instruction) VisitSend(instr *ssa.Send) {
	v.MiGo.AddStmts(migoSend(v, instr.Chan, v.Get(instr.Chan)))
}

func (v *Instruction) VisitSlice(instr *ssa.Slice) {
	handle := v.Get(instr.X)
	if instr.Low == nil && instr.High == nil { // Full slice.
		v.Put(instr, handle)
	}
}

func (v *Instruction) VisitStore(instr *ssa.Store) {
	val := v.Get(instr.Val)
	if val != nil {
		v.Put(instr.Addr, val)
	} else {
		v.Logger.Fatalf("Store: %s is not defined", instr.Val.Name())
	}
}

func (v *Instruction) VisitTypeAssert(instr *ssa.TypeAssert) {
	v.Put(instr, v.Get(instr.X))
}

func (v *Instruction) VisitUnOp(instr *ssa.UnOp) {
	switch instr.Op {
	case token.ARROW:
		v.MiGo.AddStmts(migoRecv(v, instr.X, v.Get(instr.X)))
	case token.MUL:
		if _, err := callctx.Deref(v.Context, instr.X, instr); err != nil {
			v.Env.Errors <- errors.WithStack(err) // internal error.
		}
	default:
		v.Logger.Debugf("%s UnOp", v.Logger.Module(), instr)
	}
}

// SetLogger sets logger for Instruction.
func (v *Instruction) SetLogger(l *Logger) {
	v.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.RedString("instr"),
	}
}

func (v *Instruction) createDefinition(c *ssa.CallCommon) *funcs.Definition {
	if !c.IsInvoke() {
		switch fn := c.Value.(type) {
		case *ssa.Function, *ssa.MakeClosure:
			def, ok := v.Get(fn).(*funcs.Definition)
			if !ok {
				def = funcs.MakeDefinition(c.StaticCallee())
				v.Put(fn, def)
			}
			v.Logger.Debugf("%s ↳ def %s", v.Logger.Module(), def.String())
			return def
		case *ssa.Builtin:
			if fn.Name() == "close" {
				if len(c.Args) != 1 {
					v.Logger.Fatal("%s inconsistent: close should have 1 arg",
						v.Logger.Module())
				}
				exported := v.FindExported(v.Context, v.Get(c.Args[0]))
				v.MiGo.AddStmts(&migo.CloseStatement{Chan: exported.Name()})
			}
			v.Logger.Debugf("%s %v", v.Logger.Module(), fn)
		}
		return nil
	}
	// Invoke mode.
	impl := c.Value // Implementation struct/object.
	implFn, err := fn.LookupMethodImpl(v.Env.Info.Prog, c.Method, impl)
	if err != nil {
		v.Logger.Infof("%s Cannot find method %v for invoke call %s",
			v.Logger.Module(), c, c.String())
		return nil // skip
	}
	def, ok := v.Get(implFn).(*funcs.Definition)
	if !ok {
		def = funcs.MakeDefinition(implFn)
		v.Put(implFn, def)
	}
	v.Logger.Debugf("%s ↳ invoke %s", v.Logger.Module(), def.String())
	return def
}

func (v *Instruction) doCall(c *ssa.Call, def *funcs.Definition) {
	call := funcs.MakeCall(def, c.Common(), c)
	v.Logger.Debugf("%s Definition: %v", v.Logger.Module(), def.String())
	v.Logger.Debugf("%s      Call: %v", v.Logger.Module(), call.String())
	fn := NewFunction(call, v.Context, v.Env)
	fn.SetLogger(v.Logger)
	v.Logger.Debugf("%s Context at caller: %s", v.Logger.Module(), v.Context)
	v.Logger.Debugf("%s Context at callee: %v", v.Logger.Module(), fn.Context)

	if len(call.Function().Blocks) == 0 {
		// Since the function does not have body,
		// calling it will not produce migo definitions.
		// Instead of trying to visit the function, skip over this.
		return
	}

	fn.EnterFunc(call.Function())
	stmt := &migo.CallStatement{Name: fn.Callee.Name()}

	v.bindCallParameters(call, fn)

	// Before adding call statement, handle return values.
	for i := range call.Parameters[call.NParam()+call.NBind():] {
		callerName := call.Return(i)
		caller := v.Get(callerName)
		callee := fn.Get(call.Definition().Return(i))
		if caller != callee {
			v.Put(callerName, callee)
			if isChan(callerName) { // Caller is a channel.
				if _, ok := callerName.(store.Unused); !ok {
					if calleeCh, ok := callee.(*chans.Chan); ok {
						v.MiGo.AddStmts(migoNewChan(callerName, calleeCh))
						v.Export(callerName) // Export caller name
					} else {
						// Callee does not initialise channel.
					}
				}
			}
		}
	}

	// Convert type Chan parameters to MiGo parameters.
	migoParams := paramsToMigoParam(v, fn, call)
	stmt.AddParams(migoParams...)
	if b, ok := fn.Analyser.(*Block); ok {
		for _, data := range b.meta {
			data.migoFunc.AddParams(migoParams...)
		}
	}
	v.MiGo.AddStmts(stmt)
}

func (v *Instruction) doGo(g *ssa.Go, def *funcs.Definition) {
	call := funcs.MakeCall(def, g.Common(), nil)
	v.Logger.Debugf("%s Definition: %v", v.Logger.Module(), def.String())
	v.Logger.Debugf("%s    Go/Call: %v", v.Logger.Module(), call.String())
	fn := NewFunction(call, v.Context, v.Env)
	fn.SetLogger(v.Logger)
	v.Logger.Debugf("%s Context at caller: %v", v.Logger.Module(), v.Context)
	v.Logger.Debugf("%s Context at callee: %v", v.Logger.Module(), fn.Context)

	fn.EnterFunc(call.Function())
	stmt := &migo.SpawnStatement{Name: fn.Callee.Name()}

	v.bindCallParameters(call, fn)

	// Convert type Chan parameters to MiGo parameters.
	migoParams := paramsToMigoParam(v, fn, call)
	stmt.AddParams(migoParams...)
	if b, ok := fn.Analyser.(*Block); ok {
		for _, data := range b.meta {
			data.migoFunc.AddParams(migoParams...)
		}
	}
	v.MiGo.AddStmts(stmt)
}

// getStruct returns the field variable and field index if the given value is a
// struct field.
func getStruct(value ssa.Value) (ssa.Value, int, bool) {
	switch value := value.(type) {
	case *ssa.FieldAddr:
		return value.X, value.Field, true
	case *ssa.UnOp:
		if value.Op == token.MUL {
			return getStruct(value.X)
		}
	}
	return nil, -1, false
}

// getParameterName looks for a parent struct or returns the origin value if it
// is not part of a struct.
func (v *Instruction) getParameterName(value ssa.Value) migo.NamedVar {
	if str, fld, ok := getStruct(value); ok {
		if s, ok := v.Get(str).(*structs.Struct); ok {
			return s.Fields[fld]
		}
	}
	// No parameter name.
	return value
}

// newChan creates a new channel instance
func (v *Instruction) newChan(ch ssa.Value) *chans.Chan {
	var bufSize int64
	bufsz, ok := ch.(*ssa.MakeChan).Size.(*ssa.Const)
	if !ok {
		v.Env.Errors <- ErrChanBufSzNonStatic{Pos: v.Env.Info.FSet.Position(ch.Pos())}
		bufSize = 1
	} else {
		bufSize = bufsz.Int64()
	}
	newch := chans.New(v.Callee, ch, bufSize)
	if updater, ok := v.Context.(callctx.Updater); ok {
		updater.PutUniq(ch, newch)
	} else {
		v.Logger.Fatal("Cannot update context")
	}
	return newch
}

const (
	selectCaseIndex = 0
	selectCaseValue = 1
)

func (v *Instruction) getSelectCases(sel *ssa.Select) migo.Statement {
	nCases := len(sel.States)
	if !sel.Blocking {
		nCases++
	}
	stmt := &migo.SelectStatement{Cases: make([][]migo.Statement, nCases)}
	var migoParams []*migo.Parameter
	for _, name := range v.Exported.names {
		migoParams = append(migoParams, &migo.Parameter{Callee: name, Caller: name})
	}
	if !sel.Blocking {
		stmt.Cases[nCases-1] = append(stmt.Cases[nCases-1], &migo.TauStatement{})
	}
	for _, selCase := range *sel.Referrers() {
		switch c := selCase.(type) {
		case *ssa.Extract:
			// Find all select-index tests.
			switch c.Index {
			case selectCaseIndex:
				for _, selTest := range *c.Referrers() {
					switch selTest := selTest.(type) {
					case *ssa.BinOp: // Select branch is this form, t_test = t_index == intval
						if con, ok := selTest.Y.(*ssa.Const); selTest.X == c && selTest.Op == token.EQL && ok {
							idx := int(con.Int64())

							bodyGuard := v.selBodyGuard(sel, idx)
							bodyBlk, defaultBlk := v.selBodyBlock(sel, idx, selTest.Block())
							if bodyBlk != nil {
								stmt.Cases[idx] = append(stmt.Cases[idx], bodyGuard)
								v.Logger.Debugf("%s Select index #%d block #%d (%s)", v.Logger.Module(), idx, bodyBlk.Index, bodyBlk.Comment)
							} else {
								v.Logger.Debugf("%s Select index #%d no continuation", v.Logger.Module(), idx)
							}
							if defaultBlk != nil {
								stmt.Cases[idx+1] = append(stmt.Cases[idx+1], migoCall(v.Callee.Name(), defaultBlk, v.Exported))
							}
							if bodyBlk != nil { // Return (no continuation)
								stmt.Cases[idx] = append(stmt.Cases[idx], migoCall(v.Callee.Name(), bodyBlk, v.Exported))
							}
						}
					default:
						v.Logger.Fatal("%s Unexpected select-index test expression",
							v.Logger.Module(), selTest.String())
					}
				}
			}
		}
	}
	return stmt
}

// selBodyGuard returns the guard action of a select case (except for default).
func (v *Instruction) selBodyGuard(sel *ssa.Select, caseIdx int) migo.Statement {
	chVar := sel.States[caseIdx].Chan
	chPos := v.Env.Info.FSet.Position(chVar.Pos())
	ch := v.Get(chVar)
	if _, ok := ch.(store.MockValue); ok {
		v.Logger.Debugf("%s Unknown channel %s.\n\t%s",
			v.Logger.Module(), ch, chPos.String())
	}
	param := v.FindExported(v.Context, ch)
	if _, isHidden := param.(Unexported); isHidden {
		v.Logger.Debugf("%s Channel %s/%s not exported in current scope.\n\t%s",
			v.Logger.Module(), sel.States[caseIdx].Chan.Name(), ch.UniqName(), chPos.String())
	}
	// Select guard actions then jump to body blocks
	switch sel.States[caseIdx].Dir {
	case types.SendOnly:
		return &migo.SendStatement{Chan: param.Name()}
	case types.RecvOnly:
		return &migo.RecvStatement{Chan: param.Name()}
	default:
		v.Logger.Fatalf("%s Select case is guarded by neither send nor receive.\n\t%s",
			v.Logger.Module(), chPos.String())
	}
	return nil
}

// selBodyBlock returns the body block of a select case and the default case if
// the block is is the last case.
//
// if tauBlk is not nil, caseIdx is guaranteed to be the last case.
func (v *Instruction) selBodyBlock(sel *ssa.Select, caseIdx int, testBlk *ssa.BasicBlock) (bodyBlk, tauBlk *ssa.BasicBlock) {
	switch inst := testBlk.Instrs[len(testBlk.Instrs)-1].(type) {
	case *ssa.If: // Normal case.
		if isLastCase := caseIdx == len(sel.States)-1 && !sel.Blocking; isLastCase {
			v.Logger.Debugf("%s Select default block #%d.\n\t%s",
				v.Logger.Module(), caseIdx+1, v.Env.getPos(sel))
			return inst.Block().Succs[0], inst.Block().Succs[1]
		}
		return inst.Block().Succs[0], nil
	case *ssa.Jump: // Else branch empty, followed by continuation of select.
		v.Logger.Debugf("%s Select default block empty (jump).\n\t%s",
			v.Logger.Module(), v.Env.getPos(sel))
		return inst.Block().Succs[0], nil
	case *ssa.Return: // Else branch empty and no continuation after select.
		v.Logger.Debugf("%s Select default block empty (return).\n\t%s",
			v.Logger.Module(), v.Env.getPos(sel))
		return nil, nil
	default:
		v.Logger.Fatalf("%s Select case has unrecognised last instruction in block.\n\t%s",
			v.Logger.Module(), v.Env.getPos(inst))
	}
	return nil, nil
}

// bindCallParameters takes a function call and matches up the definition
// paramters with the call arguments.
func (v *Instruction) bindCallParameters(call *funcs.Call, fn *Function) {
	handleNilChanArg := func(arg, param store.Key) {
		v.Logger.Infof("%s Handle nilchan parameter: %s=%#v, %s=%v",
			v.Logger.Module(), arg.Name(), arg, param.Name(), param)
		switch calleeChan := fn.Get(param).(type) {
		case *chans.Chan:
			v.MiGo.AddStmts(migoNewChan(arg, calleeChan))
			v.Export(arg)
		case store.MockValue:
			// Unchanged.
		}
	}

	// Before adding call statement, handle nil parameters.
	for i := range call.Parameters[:call.NParam()+call.NBind()] {
		arg, param := call.Param(i), call.Definition().Param(i)
		if isStruct(arg) {
			argStruct := v.Get(arg)
			paramStruct := fn.Get(param)
			if mock, ok := argStruct.(store.MockValue); ok {
				v.Logger.Debugf("%s %s is a nil struct (arg) (type:%s)",
					v.Logger.Module(), arg.Name(), arg.Type().String())
				argStruct = structs.New(mock, arg.(ssa.Value))
			} else if _, ok := argStruct.(*structs.Struct); !ok {
				argStruct = structs.New(mock, arg.(ssa.Value))
			}
			if mock, ok := paramStruct.(store.MockValue); ok {
				v.Logger.Debugf("%s %s is a nil struct (param) (type:%s)",
					v.Logger.Module(), param.Name(), param.Type().String())
				paramStruct = structs.New(mock, param.(ssa.Value))
			} else if _, ok := paramStruct.(*structs.Struct); !ok {
				paramStruct = structs.New(mock, arg.(ssa.Value))
			}
			argFields := argStruct.(*structs.Struct).Expand()
			paramFields := paramStruct.(*structs.Struct).Expand()
			for i := 0; i < len(argFields); i++ {
				switch argField := argFields[i].(type) {
				case structs.SField:
					paramField := paramFields[i].(structs.SField)
					if isChan(argFields[i]) {
						// This is really nil and not MockValue.
						if argField.Key == nil && paramField.Key != nil {
							handleNilChanArg(argField, paramFields[i].(structs.SField).Key) // Use actual param.
						}
					}
					// Field defined inside function, add defined value and
					// update struct.Fields
					if argField.Key == nil && paramField.Key != nil {
						v.Put(argField, fn.Get(paramField.Key))
						argField.Struct.Fields[argField.Index] = argField
					}
				case *structs.Struct:
					// Ignore.
				}
			}
		}
		if isChan(arg) {
			if _, ok := v.Get(arg).(store.MockValue); ok {
				if _, isPhi := arg.(*ssa.Phi); !isPhi {
					handleNilChanArg(arg, param)
				}
			}
		}
	}
}
