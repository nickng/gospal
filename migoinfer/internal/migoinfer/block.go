package migoinfer

import (
	"fmt"
	"go/token"

	"github.com/fatih/color"
	"github.com/nickng/gospal/block"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/loop"
	"github.com/nickng/migo"
	"golang.org/x/tools/go/ssa"
)

type BlockData struct {
	visitNode *block.VisitNode
	migoFunc  *migo.Function
	emitted   bool // Ensures if-block only gets 1 MiGo statement.
}

// Block is an analyser of ssa.BasicBlock.
type Block struct {
	*block.VisitGraph
	data []*BlockData

	Callee          *funcs.Instance // Instance of this function.
	callctx.Context                 // Function context.
	Env             *Environment    // Program environment.

	Loop      *loop.Detector // Loop detector.
	*Exported                // Local variables.
	*Logger
}

func NewBlock(fn *funcs.Instance, ctx callctx.Context, env *Environment) *Block {
	nBlk := len(fn.Function().Blocks)
	if nBlk == 0 {
		return nil // No SSA function body.
	}
	blks := make([]*BlockData, nBlk)
	for i := range blks {
		if i == 0 { // Block 0: entry (no index in name).
			blks[i] = &BlockData{
				visitNode: block.NewVisitNode(fn.Function().Blocks[i]),
				migoFunc:  migo.NewFunction(fn.Name()),
			}
		} else {
			blks[i] = &BlockData{
				visitNode: block.NewVisitNode(fn.Function().Blocks[i]),
				migoFunc:  migo.NewFunction(fmt.Sprintf("%s#%d", fn.Name(), i)),
			}
		}
	}
	b := Block{
		VisitGraph: block.NewVisitGraph(false),
		data:       blks,
		Callee:     fn,
		Context:    ctx,
		Env:        env,
		Loop:       loop.NewDetector(),
	}
	return &b
}

func (b *Block) EnterBlk(blk *ssa.BasicBlock) {
	b.Logger.Debugf("%s Enter %s#%d",
		b.Logger.Module(), b.Callee.UniqName(), blk.Index)
	for _, name := range b.Exported.names {
		b.data[blk.Index].migoFunc.AddParams(&migo.Parameter{Callee: name, Caller: name})
	}
	if !b.Visited(b.data[blk.Index].visitNode) {
		b.Visit(b.data[blk.Index].visitNode)
		b.visitInstrs(blk)
	}
}

func (b *Block) JumpBlk(curr *ssa.BasicBlock, next *ssa.BasicBlock) {
	b.Loop.Detect(curr, next)
	b.Logger.Debugf("%s Jump %s#%d → %d",
		b.Logger.Module(), b.Callee.UniqName(), curr.Index, next.Index)
	for _, name := range b.Exported.names {
		b.data[next.Index].migoFunc.AddParams(&migo.Parameter{Callee: name, Caller: name})
	}
	bnCurr, bnNext := b.data[curr.Index].visitNode, b.data[next.Index].visitNode
	if !b.Visited(bnNext) {
		// Marks the bnCurr -> bnNext edge visited.
		b.VisitFrom(bnCurr, bnNext)
		if bnCurr.Index() == bnNext.Index() && b.Visited(bnNext) {
		} else {
			b.visitInstrs(next)
		}
	}
}

func (b *Block) ExitBlk(blk *ssa.BasicBlock) {
	b.Logger.Debugf("%s Exit %s#%d",
		b.Logger.Module(), b.Callee.UniqName(), blk.Index)
}

func (b *Block) CurrBlk() *ssa.BasicBlock {
	return b.LastNode().Blk()
}

func (b *Block) PrevBlk() *ssa.BasicBlock {
	if b.Size() < 1 {
		b.Logger.Warnf("%s Cannot find PrevBlk: %#v", b.Logger.Module(), b.LastNode())
		return nil
	}
	return b.LastNode().Prev.Blk()
}

// SetLogger sets logger for Block.
func (b *Block) SetLogger(l *Logger) {
	b.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.GreenString("block"),
	}
}

// visitInstrs traverses the instructions inside the (unvisited) block.
func (b *Block) visitInstrs(blk *ssa.BasicBlock) {
	blkData := b.data[blk.Index]

	// Create a new instruction visitor for a new MiGo function.
	blkBody := NewInstruction(b.Callee, b.Context, b.Env, blkData.migoFunc)
	blkBody.Exported = b.Exported
	blkBody.SetLogger(b.Logger)
	// Handle control-flow instructions.
	for _, instr := range blk.Instrs {
		switch instr := instr.(type) { // These should be at the end of the blocks.
		case *ssa.Jump:
			blkBody.VisitJump(instr)
			call := migoCall(b.Callee.Name(), blk.Succs[0], b.Exported)
			// JumpBlk rewrites parameter so has to come after call.
			b.JumpBlk(blk, blk.Succs[0])
			blkData.migoFunc.AddStmts(call)

		case *ssa.If:
			blkBody.VisitIf(instr)
			b.Loop.ExtractCond(instr)
			b.JumpBlk(blk, blk.Succs[0])
			b.JumpBlk(blk, blk.Succs[1])
			// Output if-then-else MiGo once.
			if b.Visited(blkData.visitNode) && !blkData.emitted {
				if l := b.Loop.ForLoopAt(blk); blk.Comment == "for.loop" && l.ParamsOK() {
					loopBody := migoCall(b.Callee.Name(), blk.Parent().Blocks[l.BodyIdx()], blkBody.Exported)
					loopDone := migoCall(b.Callee.Name(), blk.Parent().Blocks[l.DoneIdx()], blkBody.Exported)
					// For loop entry block.
					iffor := &migo.IfForStatement{
						ForCond: l.String(),
						Then:    []migo.Statement{loopBody},
						Else:    []migo.Statement{loopDone},
					}
					blkData.migoFunc.AddStmts(iffor)
					blkData.emitted = true
				} else if isSelCondBlk(instr.Cond) {
					// Select case body block.
					blkData.emitted = true
				} else if blk.Comment != "cond.true" && blk.Comment != "cond.false" {
					callThen := migoCall(b.Callee.Name(), blk.Succs[0], blkBody.Exported)
					callElse := migoCall(b.Callee.Name(), blk.Succs[1], blkBody.Exported)
					// For loop intermediate blocks.
					ifstmt := &migo.IfStatement{
						Then: []migo.Statement{callThen},
						Else: []migo.Statement{callElse},
					}
					blkData.migoFunc.AddStmts(ifstmt)
					blkData.emitted = true
				}
			}

		case *ssa.Return:
			if b.Visited(blkData.visitNode) {
				blkBody.VisitReturn(instr)
			}
			b.ExitBlk(blk)

		case *ssa.Call:
			if b.Visited(blkData.visitNode) {
				b.Logger.Debugf("%s ---- CALL ---- #%d\n\t%s",
					b.Logger.Module(), blkData.visitNode.Index(), b.Env.getPos(instr))
				blkBody.VisitCall(instr)
			}

		case *ssa.Go:
			if b.Visited(blkData.visitNode) {
				b.Logger.Debugf("%s ---- SPAWN ---- #%d\n\t%s",
					b.Logger.Module(), blkData.visitNode.Index(), b.Env.getPos(instr))
				blkBody.VisitGo(instr)
			}

		case *ssa.Phi:
			blkBody.VisitPhi(instr)
			b.mergePhi(blkData, instr)
			b.Loop.ExtractIndex(instr)

		default:
			blkBody.VisitInstr(instr)
		}
	}
}

// isSelCondBlk returns true if cond is a select-state test block boolean.
func isSelCondBlk(cond ssa.Value) bool {
	if binop, ok := cond.(*ssa.BinOp); ok && binop.Op == token.EQL {
		if ext, ok := binop.X.(*ssa.Extract); ok && ext.Index == 0 {
			if _, ok := ext.Tuple.(*ssa.Select); ok {
				return true
			}
		}
	}
	return false
}

// mergePhi deals with variables in the context and exported names for φ.
//
// Given a φ-node, e.g.
//   t6 = φ[0: t1, 1: t2]
// t6 is removed from the function parameter and call argument.
// Variable from the incoming edge, e.g. 0 → t1 is located from the args then
// its corresponding parameter at callee is replaced by t6.
// The original context, e.g.
//   [ t1 → a, ... ]
// Is then converted to use the φ name, i.e. t6
//  [ t6 → a, t1 → a...]
// The original name is unexported (callee no longer have access),
// but the new φ name is exported (callee will call old name with new name).
//
func (b *Block) mergePhi(data *BlockData, instr *ssa.Phi) {
	migoFn := data.migoFunc
	removed := 0
	b.Logger.Debugf("%s Remove φ argument %s", b.Logger.Module(), instr.Name())
	for i := 0; i < len(migoFn.Params); i++ {
		if migoFn.Params[i-removed].Caller.Name() == instr.Name() || migoFn.Params[i-removed].Callee.Name() == instr.Name() {
			migoFn.Params = append(migoFn.Params[:i-removed], migoFn.Params[i-removed+1:]...)
			removed++
		}
	}
	var edge ssa.Value
	for i, pred := range data.visitNode.Blk().Preds {
		if pred.Index == data.visitNode.Prev.Index() {
			edge = instr.Edges[i]
		}
	}
	// If edge is in SSA do replace edges and parameters.
	if edge != nil {
		b.Logger.Debugf("%s Replace φ edges %s with %s in parameter",
			b.Logger.Module(), edge.Name(), instr.Name())
		for i := range migoFn.Params {
			if edge.Name() == migoFn.Params[i].Caller.Name() {
				// Update def parameters.
				migoFn.Params[i].Callee = instr
				// Update context.
				b.Context.Put(instr, b.Context.Get(edge))
				// Update exported names.
				b.Unexport(edge)
				b.Export(instr)
			}
		}
	}
}
