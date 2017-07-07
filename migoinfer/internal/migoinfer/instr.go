package migoinfer

import (
	"github.com/fatih/color"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/fn"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/store"
	"github.com/nickng/migo"
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
}

func (v *Instruction) VisitAlloc(instr *ssa.Alloc) {
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
}

func (v *Instruction) VisitMakeClosure(instr *ssa.MakeClosure) {
	def := funcs.MakeClosureDefinition(instr.Fn.(*ssa.Function), instr.Bindings)
	v.Put(instr, def)    // For calling the closure.
	v.Put(instr.Fn, def) // For reusing the closure.
	f := v.Get(instr)
	v.Logger.Debugf("%s ↳ %s", v.Logger.Module(), f.(*funcs.Definition).String())
}

func (v *Instruction) VisitMakeInterface(instr *ssa.MakeInterface) {
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
}

func (v *Instruction) VisitSend(instr *ssa.Send) {
}

func (v *Instruction) VisitSlice(instr *ssa.Slice) {
}

func (v *Instruction) VisitStore(instr *ssa.Store) {
}

func (v *Instruction) VisitTypeAssert(instr *ssa.TypeAssert) {
}

func (v *Instruction) VisitUnOp(instr *ssa.UnOp) {
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

	// Convert type Chan parameters to MiGo parameters.
	migoParams := paramsToMigoParam(v, fn, call)
	stmt.AddParams(migoParams...)
	if b, ok := fn.Analyser.(*Block); ok {
		for _, data := range b.data {
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

	// Convert type Chan parameters to MiGo parameters.
	migoParams := paramsToMigoParam(v, fn, call)
	stmt.AddParams(migoParams...)
	if b, ok := fn.Analyser.(*Block); ok {
		for _, data := range b.data {
			data.migoFunc.AddParams(migoParams...)
		}
	}
	v.MiGo.AddStmts(stmt)
}
