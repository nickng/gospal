package migoinfer

import (
	"github.com/fatih/color"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/migo"
	"golang.org/x/tools/go/ssa"
)

// Instruction is a visitor for related instructions within a block.
type Instruction struct {
	Callee          *funcs.Instance // Instance of this function.
	callctx.Context                 // Function context.
	Env             *Environment    // Program environment.

	MiGo *migo.Function // MiGo function definition of current block.
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
