package loop

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// Info is a data structure to hold loop information,
// for tracking condition variables, index variable.
type Info struct {
	indexVar ssa.Value // Variable holding the index (phi).

	condRoot *BinTree               // Variable holding root cond expression.
	subtrees map[ssa.Value]*BinTree // Quick lookup for subtree.
	prevCond ssa.Value

	initVal int64 // initial value.
	stepVal int64 // step value.

	loopIdx int // Block index of for.loop.
	bodyIdx int // Block index of for.body.
	doneIdx int // Block index of for.done.

	indexOK, condOK bool // Sanity check to ensure index/condition is valid.

	target *BinTree
}

func New(index int) *Info {
	return &Info{
		loopIdx:  index,
		subtrees: make(map[ssa.Value]*BinTree),
		target:   &BinTree{Target: true},
	}
}

// SetBodyBlock sets the body block index of the for-loop.
func (i *Info) SetBodyBlock(index int) {
	i.bodyIdx = index
}

// SetDoneBlock sets the done block index of the for-loop.
func (i *Info) SetDoneBlock(index int) {
	i.doneIdx = index
}

// SetCond is used for setting the first (i.e., 'root') condition.
func (i *Info) SetCond(cond ssa.Value) {
	if i.condRoot == nil {
		i.condRoot = &BinTree{Cond: cond}
		i.subtrees[cond] = i.condRoot
	}
}

// SetParentCond is for adjusting the BinTree so it points to the correct
// conditional subtree root.
func (i *Info) SetParentCond(prevCond ssa.Value) {
	i.prevCond = prevCond
}

func (i *Info) AddTrue(cond ssa.Value) {
	if prev, exists := i.subtrees[i.prevCond]; exists {
		if prev.True == nil {
			if t, exists := i.subtrees[cond]; exists {
				// If a node is previously visited, this makes a cycle.
				prev.True = t
			} else {
				prev.True = &BinTree{Cond: cond}
				i.subtrees[cond] = prev.True
			}
		}
		return
	}
	log.Fatal("AddTrue to subtree that does not exist")
}

func (i *Info) AddFalse(cond ssa.Value) {
	if prev, exists := i.subtrees[i.prevCond]; exists {
		if prev.False == nil {
			if t, exists := i.subtrees[cond]; exists {
				// If a node is previously visited, this makes a cycle.
				prev.False = t
			} else {
				prev.False = &BinTree{Cond: cond}
				i.subtrees[cond] = prev.False
			}
		}
		return
	}
	log.Fatal("AddFalse to subtree that does not exist")
}

// MarkTarget points a part of a loop condition to an existing subtree.
func (i *Info) MarkTarget() {
	if prev, exists := i.subtrees[i.prevCond]; exists {
		prev.True = i.target
		return
	}
	log.Fatal("MarkTarget to subtree that does not exist")
}

func binTreeToString(t *BinTree) []string {
	var exprs []string

	if t.True != nil && t.True.Target {
		return []string{t.String()}
	}

	if t.True != nil {
		exprs = append(exprs, binTreeToString(t.True)...)
	}

	if t.False != nil {
		exprs = append(exprs, binTreeToString(t.False)...)
	}

	return exprs
}

func (i *Info) String() string {
	var buf bytes.Buffer
	if i.indexVar != nil {
		buf.WriteString(fmt.Sprintf("%s = %d; ", i.indexVar.Name(), i.initVal))
		if i.condRoot != nil {
			btv := NewBinTreeVisitor()
			btv.VisitRoot(i.condRoot)
			buf.WriteString(fmt.Sprintf("%s; ", strings.Join(binTreeToString(i.condRoot), "||")))
		}
		if i.stepVal > 0 {
			buf.WriteString(fmt.Sprintf("%s = %s + %d", i.indexVar.Name(), i.indexVar.Name(), i.stepVal))
		} else {
			buf.WriteString(fmt.Sprintf("%s = %s - %d", i.indexVar.Name(), i.indexVar.Name(), -i.stepVal))
		}
	}
	return buf.String()
}

func (i *Info) BodyIdx() int { return i.bodyIdx }

func (i *Info) DoneIdx() int { return i.doneIdx }

// ParamsOK returns true iff both index and cond are detected correctly.
func (i *Info) ParamsOK() bool {
	return i.indexVar != nil && i.indexOK && i.condRoot != nil && i.condRoot.Cond != nil && i.condOK
}
