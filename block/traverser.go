package block

import (
	"golang.org/x/tools/go/ssa"
)

// TraverseEdges takes a Function and apply visit to each edge.
func TraverseEdges(fn *ssa.Function, visit func(from, to *ssa.BasicBlock)) {
	visited := NewVisitGraph(false)
	if len(fn.Blocks) == 0 {
		return
	}
	type Edge struct {
		From, To *ssa.BasicBlock
	}
	queue := []Edge{Edge{To: fn.Blocks[0]}}
	for len(queue) > 0 {
		e := queue[0]
		queue = queue[1:]
		if !visited.NodeVisited(NewVisitNode(e.To)) {
			if e.From == nil {
				visited.Visit(NewVisitNode(e.To))
			} else {
				visited.VisitFrom(NewVisitNode(e.From), NewVisitNode(e.To))
			}
			visit(e.From, e.To)
			for _, succ := range e.To.Succs {
				queue = append(queue, Edge{From: e.To, To: succ})
			}
		}
	}
}
