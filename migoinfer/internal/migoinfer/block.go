package migoinfer

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/nickng/gospal/block"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/migo"
	"golang.org/x/tools/go/ssa"
)

type BlockData struct {
	visitNode *block.VisitNode
	migoFunc  *migo.Function
}

// Block is an analyser of ssa.BasicBlock.
type Block struct {
	*block.VisitGraph
	data []*BlockData

	Callee          *funcs.Instance // Instance of this function.
	callctx.Context                 // Function context.
	Env             *Environment    // Program environment.

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
	}
	return &b
}

func (b *Block) EnterBlk(blk *ssa.BasicBlock) {
}

func (b *Block) JumpBlk(curr *ssa.BasicBlock, next *ssa.BasicBlock) {
}

func (b *Block) ExitBlk(blk *ssa.BasicBlock) {
}

func (b *Block) CurrBlk() *ssa.BasicBlock {
	return nil
}

func (b *Block) PrevBlk() *ssa.BasicBlock {
	return nil
}

// SetLogger sets logger for Block.
func (b *Block) SetLogger(l *Logger) {
	b.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.GreenString("block"),
	}
}
