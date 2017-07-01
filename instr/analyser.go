// Package instr provides the Analyser interface for instructions.
package instr

import "golang.org/x/tools/go/ssa"

// Analyser is an interface for Instruction analysis,
// handles each defined Instruction.
type Analyser interface {
	VisitInstr(instr ssa.Instruction)
	VisitAlloc(instr *ssa.Alloc)
	VisitBinOp(instr *ssa.BinOp)
	VisitCall(instr *ssa.Call)
	VisitChangeInterface(instr *ssa.ChangeInterface)
	VisitChangeType(instr *ssa.ChangeType)
	VisitConvert(instr *ssa.Convert)
	VisitDebugRef(instr *ssa.DebugRef)
	VisitDefer(instr *ssa.Defer)
	VisitExtract(instr *ssa.Extract)
	VisitField(instr *ssa.Field)
	VisitFieldAddr(instr *ssa.FieldAddr)
	VisitGo(instr *ssa.Go)
	VisitIf(instr *ssa.If)
	VisitIndex(instr *ssa.Index)
	VisitIndexAddr(instr *ssa.IndexAddr)
	VisitJump(instr *ssa.Jump)
	VisitLookup(instr *ssa.Lookup)
	VisitMakeChan(instr *ssa.MakeChan)
	VisitMakeClosure(instr *ssa.MakeClosure)
	VisitMakeInterface(instr *ssa.MakeInterface)
	VisitMakeMap(instr *ssa.MakeMap)
	VisitMakeSlice(instr *ssa.MakeSlice)
	VisitMapUpdate(instr *ssa.MapUpdate)
	VisitNext(instr *ssa.Next)
	VisitPanic(instr *ssa.Panic)
	VisitPhi(instr *ssa.Phi)
	VisitRange(instr *ssa.Range)
	VisitReturn(instr *ssa.Return)
	VisitRunDefers(instr *ssa.RunDefers)
	VisitSelect(instr *ssa.Select)
	VisitSend(instr *ssa.Send)
	VisitSlice(instr *ssa.Slice)
	VisitStore(instr *ssa.Store)
	VisitTypeAssert(instr *ssa.TypeAssert)
	VisitUnOp(instr *ssa.UnOp)
}
