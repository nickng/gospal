// package block provides the Analyser interface for blocks and supporting utils.
package block

import "golang.org/x/tools/go/ssa"

// Analyser is an interface for BasicBlock analysis,
// handles block transitions within functions.
type Analyser interface {
	// EnterBlk analyses a block where there is no predecessor or just follow
	// the natural order,
	// e.g. the first block in a Function, or the direct predecessor.
	EnterBlk(blk *ssa.BasicBlock)

	// JumpBlk analyses a block where the predecessor is specified explicitly,
	// where the transfer of control may impact the control flow directly.
	JumpBlk(curr, next *ssa.BasicBlock)

	// ExitBlk analyses a terminating block where there are no successors, this
	// should be called when marking a block an end.
	// Typically this is called in the end of a block where it was entered via
	// EnterBlk or JumpBlk, so blk should not be "visited".
	ExitBlk(blk *ssa.BasicBlock)

	// CurrBlk returns the current block (last block entered).
	CurrBlk() *ssa.BasicBlock

	// PrevBlk() returns the previous block (last block exited).
	PrevBlk() *ssa.BasicBlock
}
