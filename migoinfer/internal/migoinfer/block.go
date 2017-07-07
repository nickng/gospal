package migoinfer

import (
	"fmt"

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
		b.visitInstrs(next)
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
			b.JumpBlk(blk, blk.Succs[0])

		case *ssa.If:
			blkBody.VisitIf(instr)
			b.Loop.ExtractCond(instr)
			b.JumpBlk(blk, blk.Succs[0])
			b.JumpBlk(blk, blk.Succs[1])

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
			b.Loop.ExtractIndex(instr)

		default:
			blkBody.VisitInstr(instr)
		}
	}
}
