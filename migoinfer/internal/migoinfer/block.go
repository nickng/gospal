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
	meta []*BlockData

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
		meta:       blks,
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
		b.meta[blk.Index].migoFunc.AddParams(&migo.Parameter{Callee: name, Caller: name})
	}
	if !b.NodeVisited(b.meta[blk.Index].visitNode) {
		b.Visit(b.meta[blk.Index].visitNode)
		b.visitInstrs(blk)
	}
}

func (b *Block) JumpBlk(curr *ssa.BasicBlock, next *ssa.BasicBlock) {
	b.Loop.Detect(curr, next)
	b.Logger.Debugf("%s Jump %s#%d → %d",
		b.Logger.Module(), b.Callee.UniqName(), curr.Index, next.Index)
	for _, name := range b.Exported.names {
		b.meta[next.Index].migoFunc.AddParams(&migo.Parameter{Callee: name, Caller: name})
	}
	blkMeta := b.meta[next.Index]
	if !b.NodeVisited(blkMeta.visitNode) {
		if !b.EdgeVisited(b.meta[curr.Index].visitNode, blkMeta.visitNode) {
			// Marks the bnCurr -> bnNext edge visited.
			b.VisitFrom(b.meta[curr.Index].visitNode, blkMeta.visitNode)
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
	blkMeta := b.meta[blk.Index]

	// Create a new instruction visitor for a new MiGo function.
	blkBody := NewInstruction(b.Callee, b.Context, b.Env, blkMeta.migoFunc)
	blkBody.Exported = b.Exported
	blkBody.SetLogger(b.Logger)
	// Handle control-flow instructions.
	for _, instr := range blk.Instrs {
		switch instr := instr.(type) { // These should be at the end of the blocks.
		case *ssa.Jump:
			blkBody.VisitJump(instr)
			if b.NodeVisited(blkMeta.visitNode) {
				blkMeta.migoFunc.AddStmts(migoCall(b.Callee.Name(), blk.Succs[0], b.Exported))
			}
			if !b.EdgeVisited(blkMeta.visitNode, b.meta[blk.Succs[0].Index].visitNode) {
				b.JumpBlk(blk, blk.Succs[0])
			}

		case *ssa.If:
			blkBody.VisitIf(instr)
			b.Loop.ExtractCond(instr)
			if !b.EdgeVisited(blkMeta.visitNode, b.meta[blk.Succs[0].Index].visitNode) {
				b.JumpBlk(blk, blk.Succs[0])
			}
			if !b.EdgeVisited(blkMeta.visitNode, b.meta[blk.Succs[1].Index].visitNode) {
				b.JumpBlk(blk, blk.Succs[1])
			}
			// Output if-then-else MiGo once.
			if b.NodeVisited(blkMeta.visitNode) && !blkMeta.emitted {
				if l := b.Loop.ForLoopAt(blk); blk.Comment == "for.loop" && l.ParamsOK() {
					loopBody := migoCall(b.Callee.Name(), blk.Parent().Blocks[l.BodyIdx()], blkBody.Exported)
					loopDone := migoCall(b.Callee.Name(), blk.Parent().Blocks[l.DoneIdx()], blkBody.Exported)
					// For loop entry block.
					iffor := &migo.IfForStatement{
						ForCond: l.String(),
						Then:    []migo.Statement{loopBody},
						Else:    []migo.Statement{loopDone},
					}
					blkMeta.migoFunc.AddStmts(iffor)
					blkMeta.emitted = true
				} else if isSelCondBlk(instr.Cond) {
					// Select case body block.
					blkMeta.emitted = true
				} else if blk.Comment != "cond.true" && blk.Comment != "cond.false" {
					callThen := migoCall(b.Callee.Name(), blk.Succs[0], blkBody.Exported)
					callElse := migoCall(b.Callee.Name(), blk.Succs[1], blkBody.Exported)
					// For loop intermediate blocks.
					ifstmt := &migo.IfStatement{
						Then: []migo.Statement{callThen},
						Else: []migo.Statement{callElse},
					}
					blkMeta.migoFunc.AddStmts(ifstmt)
					blkMeta.emitted = true
				}
			}

		case *ssa.Return:
			if b.NodeVisited(blkMeta.visitNode) {
				blkBody.VisitReturn(instr)
			}
			b.ExitBlk(blk)

		case *ssa.Call:
			if b.NodeVisited(blkMeta.visitNode) {
				b.Logger.Debugf("%s ---- CALL ---- #%d\n\t%s",
					b.Logger.Module(), blkMeta.visitNode.Index(), b.Env.getPos(instr))
				blkBody.VisitCall(instr)
			}

		case *ssa.Go:
			if b.NodeVisited(blkMeta.visitNode) {
				b.Logger.Debugf("%s ---- SPAWN ---- #%d\n\t%s",
					b.Logger.Module(), blkMeta.visitNode.Index(), b.Env.getPos(instr))
				blkBody.VisitGo(instr)
			}

		case *ssa.Phi:
			blkBody.VisitPhi(instr)
			b.mergePhi(blkMeta, instr)
			b.Loop.ExtractIndex(instr)

		default:
			if b.NodeVisited(blkMeta.visitNode) {
				blkBody.VisitInstr(instr)
			}
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
