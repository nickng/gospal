package loop

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type BinTreeVisitor struct {
	toVisit []*BinTree
	visited map[*BinTree]bool
}

func NewBinTreeVisitor() *BinTreeVisitor {
	return &BinTreeVisitor{
		visited: make(map[*BinTree]bool),
	}
}

func (v *BinTreeVisitor) VisitRoot(t *BinTree) {
	v.toVisit = append(v.toVisit, t)
	v.CalcPrefix()
}

// CalcPrefix propagates all conditions prefix to the 'prefix' so that String()
// will give the string representation of the whole conditional expression.
func (v *BinTreeVisitor) CalcPrefix() {
	if len(v.toVisit) > 0 {
		t := v.toVisit[0]
		v.toVisit = v.toVisit[1:]
		if _, ok := v.visited[t]; ok {
			// If this is an update.
			if t.True != nil {
				if t.True.Target {
					for i, p := range t.True.prefix {
						strings.Contains(t.TrueString(), p)
						t.True.prefix[i] = t.TrueString()
					}
				} else {
					t.True.prefix = append(t.True.prefix, t.TrueString())
				}
				v.toVisit = append(v.toVisit, t.True)
			}
		} else {
			if t.True != nil {
				t.True.prefix = append(t.True.prefix, t.TrueString())
				v.toVisit = append(v.toVisit, t.True)
			}
		}
		if t.False != nil {
			t.False.prefix = append(t.False.prefix, t.FalseString())
			v.toVisit = append(v.toVisit, t.False)
		}
		v.visited[t] = true
		v.CalcPrefix()
	}
}

// BinTree is a binary tree for conditions.
type BinTree struct {
	prefix []string
	Cond   ssa.Value
	True   *BinTree
	False  *BinTree

	Target bool
}

func (t *BinTree) SetTrue(c ssa.Value)  { t.True = &BinTree{Cond: c} }
func (t *BinTree) SetFalse(c ssa.Value) { t.False = &BinTree{Cond: c} }

func (t *BinTree) String() string {
	return t.TrueString()
}

func (t *BinTree) TrueString() string {
	if t.Target {
		return "TARGET"
	}
	var buf bytes.Buffer
	this := exprToString(t.Cond)
	if len(t.prefix) > 0 {
		for i, prefix := range t.prefix {
			if i > 0 {
				buf.WriteString("||")
			}
			buf.WriteString(fmt.Sprintf("(%s && %s)", prefix, this))
		}
	} else {
		buf.WriteString(this)
	}
	return buf.String()
}

func (t *BinTree) FalseString() string {
	var buf bytes.Buffer
	this := exprToString(t.Cond)
	if len(t.prefix) > 0 {
		for i, prefix := range t.prefix {
			if i > 0 {
				buf.WriteString("||")
			}
			buf.WriteString(fmt.Sprintf("(%s && !%s)", prefix, this))
		}
	} else {
		buf.WriteString(fmt.Sprintf("!%s", this))
	}
	return buf.String()
}

// exprToString converts a conditional expression to string.
func exprToString(expr ssa.Value) string {
	switch expr := expr.(type) {
	case *ssa.Phi:
		return expr.Name()
	case *ssa.Const:
		if !expr.IsNil() && expr.Value.Kind() == constant.Int {
			return fmt.Sprintf("%d", expr.Int64())
		} else {
			return expr.String() // not supported
		}
	case *ssa.BinOp:
		switch expr.Op {
		case token.ADD:
			return fmt.Sprintf("(%s+%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.SUB:
			return fmt.Sprintf("(%s-%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.MUL:
			return fmt.Sprintf("(%s*%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.QUO:
			return fmt.Sprintf("(%s/%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.REM:
			return fmt.Sprintf("(%s%%%s)", exprToString(expr.X), exprToString(expr.Y))

		case token.EQL:
			return fmt.Sprintf("(%s==%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.LSS:
			return fmt.Sprintf("(%s<%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.GTR:
			return fmt.Sprintf("(%s>%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.NEQ:
			return fmt.Sprintf("(%s!=%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.LEQ:
			return fmt.Sprintf("(%s<=%s)", exprToString(expr.X), exprToString(expr.Y))
		case token.GEQ:
			return fmt.Sprintf("(%s>=%s)", exprToString(expr.X), exprToString(expr.Y))

		default:
			return expr.Name() // not supported
		}
	case *ssa.UnOp:
		switch expr.Op {
		case token.SUB:
			return fmt.Sprintf("(-%s)", exprToString(expr.X))
		default:
			return expr.Name() // not supported
		}
	default:
		return expr.Name() // not supported
	}
}
