package loop

import (
	"errors"
	"fmt"
	"go/constant"
	"go/token"
	"io"
	"io/ioutil"
	"log"

	"golang.org/x/tools/go/ssa"
)

var ErrIdxNotInt = errors.New("index is not int")

// State is the loop transition states.
type State int

const (
	NonLoop State = iota
	Enter
	CondTrue  // CondTrue is an extension of Enter for complex conditions
	CondFalse // CondFalse is an extensions of Enter for complex conditions
	Body
	Exit
)

type Detector struct {
	scope    *Stack
	loopInfo map[int]*Info // loopInfo holds info (e.g. for nested loop) as traversal is non-linear (so cannot use stack)
	logger   *log.Logger

	blockState map[*ssa.BasicBlock]State
	blockScope map[*ssa.BasicBlock]*Info
}

func NewDetector() *Detector {
	return &Detector{
		scope:    NewStack(),
		loopInfo: make(map[int]*Info),
		logger:   log.New(ioutil.Discard, "loopdetect: ", 0),

		blockState: make(map[*ssa.BasicBlock]State),
		blockScope: make(map[*ssa.BasicBlock]*Info),
	}
}

func (d *Detector) SetLog(w io.Writer) {
	d.logger.SetOutput(w)
}

// ExtractIndex takes a Phi inside an Enter state work out the initial value
// and increment.
func (d *Detector) ExtractIndex(phi *ssa.Phi) {
	state, exists := d.blockState[phi.Block()]
	if !exists {
		d.logger.Printf("No condition to extract at unknown state")
		return
	}
	if state != Enter {
		return
	}
	scope, exists := d.blockScope[phi.Block()]
	if !exists {
		d.logger.Printf("ExtractIndex: #%d is not part of a loop", phi.Block().Index)
		return
	}

	for i := 0; i < 2; i++ {
		switch edge := phi.Edges[i].(type) {
		case *ssa.Const:
			if val, err := getIntConst(edge); err == nil {
				scope.initVal = val
				scope.indexVar = phi
			}
		case *ssa.BinOp:
			switch edge.Op {
			case token.ADD:
				if y, ok := edge.Y.(*ssa.Const); ok {
					if val, err := getIntConst(y); err == nil {
						scope.stepVal = val
						scope.indexOK = true
					}
				}
			case token.SUB:
				if y, ok := edge.Y.(*ssa.Const); ok {
					if val, err := getIntConst(y); err == nil {
						scope.stepVal = -val
						scope.indexOK = true
					}
				}
			}
		}
	}
}

func (d *Detector) ExtractCond(ifelse *ssa.If) {
	state, exists := d.blockState[ifelse.Block()]
	if !exists {
		d.logger.Printf("No condition to extract at unknown state")
		d.blockState[ifelse.Block()] = NonLoop
		return
	}
	scope, exists := d.blockScope[ifelse.Block()]
	if !exists {
		d.logger.Printf("ExtractCond: #%d is not part of a loop", ifelse.Block().Index)
		return
	}
	d.logger.Printf("ExtractCond #%d: Cond: %s", ifelse.Block().Index, ifelse.Cond.String())

	switch state {
	case Enter: // This is the root condition.
		scope.SetCond(ifelse.Cond)
		// Now run sanity check.
		if scope.indexVar != nil {
			// condition must involve indexVar
			if usesIndexVar(scope.condRoot.Cond, scope.indexVar) {
				scope.condOK = true
			}
		}
	case CondTrue: // Intermediate condition.
		scope.AddTrue(ifelse.Cond)
	case CondFalse: // Intermediate condition.
		scope.AddFalse(ifelse.Cond)
	}
}

func (d *Detector) Detect(from, to *ssa.BasicBlock) {
	if _, exists := d.blockState[from]; !exists {
		d.blockState[from] = NonLoop
	}
	d.logger.Printf("Detect: #%d → #%d", from.Index, to.Index)
	switch d.blockState[from] {
	case NonLoop:
		switch to.Comment {
		case "for.loop":
			d.logger.Printf("NonLoop (#%d) → Enter (#%d)", from.Index, to.Index)
			d.blockState[to] = Enter
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = New(to.Index)
				d.logger.Printf("NonLoop → Enter: registers new loop at #%d", to.Index)
			}
		}
	case Enter, CondTrue, CondFalse: // This block has loop condition.
		// Because the parent is also an if-else, they end with *ssa.If.
		parentCond := from.Instrs[len(from.Instrs)-1].(*ssa.If).Cond
		switch to.Comment {
		case "for.body":
			d.logger.Printf("Enter/Cond (#%d) → Body (#%d)", from.Index, to.Index)
			d.blockState[to] = Body
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = d.blockScope[from]
				d.blockScope[to].bodyIdx = to.Index
				d.logger.Printf("Enter/Cond → Body: copy scope from #%d", from.Index)
			}
			d.blockScope[to].SetParentCond(parentCond)
			d.blockScope[to].MarkTarget()
		case "for.done":
			d.logger.Printf("Enter/Cond (#%d) → Done (#%d)", from.Index, to.Index)
			d.blockState[to] = Exit
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = d.blockScope[from]
				d.blockScope[to].doneIdx = to.Index
				d.logger.Printf("Enter/Cond → Done: copy scope from #%d", from.Index)
			}
			d.blockScope[to].SetParentCond(parentCond)
		case "cond.true":
			d.logger.Printf("Enter/Cond (#%d) → Cond|True (#%d)", from.Index, to.Index)
			d.blockState[to] = CondTrue
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = d.blockScope[from]
				d.logger.Printf("Enter/Cond → Cond|True: copy scope from #%d", from.Index)
			}
			d.blockScope[to].SetParentCond(parentCond)
		case "cond.false":
			d.logger.Printf("Enter/Cond (#%d) → Cond|False (#%d)", from.Index, to.Index)
			d.blockState[to] = CondFalse
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = d.blockScope[from]
				d.logger.Printf("Enter/Cond → Cond|False: copy scope from #%d", from.Index)
			}
			d.blockScope[to].SetParentCond(parentCond)
		}
	case Body: // Body code, may have nesting.
		switch to.Comment {
		case "for.loop":
			d.logger.Printf("Body (#%d) → Enter (#%d): Nesting", from.Index, to.Index)
			d.blockState[to] = Enter
			if _, exists := d.blockScope[to]; !exists {
				d.blockScope[to] = New(to.Index)
				d.logger.Printf("Body → Enter: registers new loop at #%d", to.Index)
			} else {
				d.logger.Printf("Body → Enter: normal loop at #%d (loop@%d)", to.Index, d.blockScope[to].loopIdx)
			}
		}
	case Exit:
		switch to.Comment {
		case "for.loop":
			d.logger.Printf("Exit (#%d) → Enter (#%d): Reenter parent/enter next loop", from.Index, to.Index)
			d.blockState[to] = Enter
			if _, exists := d.blockScope[to]; !exists {
				d.logger.Printf("Exit → Enter: New loop #%d (after loop@%d)", to.Index, d.blockScope[from].loopIdx)
				d.blockScope[to] = New(to.Index)
			}
		default:
			d.logger.Printf("Exit (#%d) → ? (#%d): '%s'", from.Index, to.Index, to.Comment)
		}
	}
}

func (d *Detector) debugShowScopes() {
	for blk, loop := range d.blockScope {
		fmt.Printf("%d: loop rooted at %d body:%t done:%t\n", blk.Index, loop.loopIdx, blk.Index == loop.bodyIdx, blk.Index == loop.doneIdx)
	}
}

func (d *Detector) ForLoopAt(b *ssa.BasicBlock) *Info {
	return d.blockScope[b]
}

// getIntConst is a helper function to extract constant int value.
func getIntConst(c *ssa.Const) (int64, error) {
	if !c.IsNil() && c.Value.Kind() == constant.Int {
		return c.Int64(), nil
	}
	return 0, ErrIdxNotInt
}

// usesIndexVar checks if the cond expression involves index.
func usesIndexVar(cond, index ssa.Value) bool {
	switch cond := cond.(type) {
	case *ssa.BinOp:
		return usesIndexVar(cond.X, index) || usesIndexVar(cond.Y, index)

	case *ssa.UnOp:
		return usesIndexVar(cond.X, index)

	default:
		return cond == index
	}
}
