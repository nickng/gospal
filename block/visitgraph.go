package block

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"golang.org/x/tools/go/ssa"
)

var (
	ErrEdgesStackEmpty = errors.New("blockgraph: cannot pop edges: stack empty")
	ErrBadNode         = errors.New("VisitNode does not contain block (or has nil block)")
	ErrBadBlock        = errors.New("internal error: Block is nil")
	ErrBadParentFn     = errors.New("internal error: Block has nil parent Fn")
)

// visitedEdges keeps track of whether blocks are visited.
//
// Edges are mapped as BasicBlock.Index --> incoming BasicBlock.Index --> bool
type visitedEdges map[int]map[int]bool

// VisitGraph is a data structure to track the control flow of execution within
// functions. Each node is the ssa.BasicBlock that the analysis has previously
// visited.
//
// VisitGraph, unlike the name suggests, is a doubly linked list.
// Traversing the VisitGraph is equivalent to going through the analysis.
type VisitGraph struct {
	sync.Mutex

	// Root is the beginning of the VisitGraph.
	// For a normal analysis, this should point the Block#0 of main.main.
	nodes []*VisitNode

	// visited keeps track of whether a block is visited.
	// visited: Function --> visitEdges
	//
	// The intuitive understanding of whether a BasicBlock is visited, is that
	// if all incoming edges (i.e. paths into a BasicBlock) have been visited.
	//
	// The visited entry of a BasicBlock is initialised with false for all
	// incoming BasicBlock.Index
	visited map[*ssa.Function]visitedEdges

	// reentrant is true if the graph keeps track of reentrant.
	reentrant bool

	// edgesStack keeps track of visited edges per function for a reentrant VisitGraph.
	edgesStack map[*ssa.Function][]visitedEdges
}

// NewVisitGraph returns a new VisitGraph.
func NewVisitGraph(reentrant bool) *VisitGraph {
	if reentrant {
		return &VisitGraph{
			visited:    make(map[*ssa.Function]visitedEdges),
			reentrant:  true,
			edgesStack: make(map[*ssa.Function][]visitedEdges),
		}
	}
	return &VisitGraph{
		visited: make(map[*ssa.Function]visitedEdges),
	}
}

// pushEdges pushes current visitedEdges to stack and clears visited for Function fn.
func (g *VisitGraph) pushEdges(fn *ssa.Function) {
	if g.reentrant {
		if edges, ok := g.visited[fn]; ok {
			g.edgesStack[fn] = append(g.edgesStack[fn], edges)
			g.visited[fn] = nil
		}
	}
}

// popEdges moves the top of visitedEdges stack to visited for Function fn.
func (g *VisitGraph) popEdges(fn *ssa.Function) error {
	if g.reentrant {
		if len(g.edgesStack[fn]) < 1 {
			return ErrEdgesStackEmpty
		}
		g.visited[fn] = g.edgesStack[fn][len(g.edgesStack[fn])-1]
		g.edgesStack[fn] = g.edgesStack[fn][:len(g.edgesStack[fn])-1]
	}
	return nil
}

// Visit enters a new BasicBlock, which adds a new VisitNode to the end of the
// VisitGraph.
//
// If the current new BasicBlock is not node 0 (entry), it is assumed that the
// immediate predecessor of the block is the last VisitNode in the VisitGraph.
func (g *VisitGraph) Visit(n *VisitNode) {
	g.Lock()
	defer g.Unlock()
	g.nodes = append(g.nodes, n)
	if len(g.nodes) > 1 {
		n.Prev = g.nodes[len(g.nodes)-2]
		g.nodes[len(g.nodes)-2].Next = n
	}

	// Initialise visited for all blocks in the Function we entered (index = 0).
	if n.Index() == 0 {
		g.markNewFuncVisit(n)
	} else {
		if n.Prev == nil {
			log.Printf("Visit: there is no previous block")
			return
		}
		g.markVisit(n.Prev, n)
	}
}

// MarkLast marks a block that has no successor VisitNode.
//
// The function has no effect in a non-reentrant VisitGraph.
//
// This assumes that the VisitNode n will not be added to the VisitGraph
// (because it is already visited). The reason this is separated from a Visit
// (which can inspect the Block and mark as 'Last') is to allow for MarkLast to
// be called in a defer.
func (g *VisitGraph) MarkLast(n *VisitNode) {
	if g.reentrant {
		g.popEdges(n.Fn())
	}
}

// markVisit marks the edge prev --> n visited.
//
// Param prev is not modified or stored, and is used for looking up the previous
// BasicBlock Index.
// Visiting a block from a different function is ignored.
func (g *VisitGraph) markVisit(prev, n *VisitNode) {
	// Jumping between functions and not at entry block: something is wrong
	if prev.Fn() != n.Fn() {
		log.Printf("markVisit: previous block %q#%d and current block %q#%d are in different function",
			prev.Fn(), prev.Index(), n.Fn(), n.Index())
		return
	}
	if _, ok := g.visited[n.Fn()]; !ok {
		log.Printf("markVisit: Function %q was not visited before but visiting block %d",
			n.Fn(), n.Index())
		g.markNewFuncVisit(n)
	}
	g.visited[n.Fn()][n.Index()][prev.Index()] = true
}

// markNewFuncVisit enters the parent Function of n.
// The visited map for the Function will be (re-)initialised.
func (g *VisitGraph) markNewFuncVisit(n *VisitNode) {
	if _, ok := g.visited[n.Fn()]; ok {
		if !g.reentrant {
			log.Printf("Function %q was visited", n.Fn())
			return
		}
		g.pushEdges(n.Fn())
		// Re-initialise all blocks in the function to be not visited.
	}
	// Initialises all blocks in the function to be not visited.
	g.visited[n.Fn()] = make(map[int]map[int]bool)
	for _, b := range n.Fn().Blocks {
		g.visited[n.Fn()][b.Index] = make(map[int]bool)
		// This iterates through all possible input edges to each block
		// and mark them not visited.
		for _, p := range b.Preds {
			g.visited[n.Fn()][b.Index][p.Index] = false
		}
	}
}

// VisitFrom is the Visit function for non-linear visits.
//
// Param prev is not modified or stored, and is used for looking up the previous
// BasicBlock Index. However, prev must be visited before.
func (g *VisitGraph) VisitFrom(prev, n *VisitNode) {
	g.Lock()
	defer g.Unlock()
	g.nodes = append(g.nodes, n)
	if len(g.nodes) > 1 {
		n.Prev = g.nodes[len(g.nodes)-2]
		g.nodes[len(g.nodes)-2].Next = n
	}

	validPrev := false
PREVCHECK:
	for i := len(g.nodes) - 1; i >= 0; i-- {
		if g.nodes[i].Blk() == prev.Blk() {
			validPrev = true
			break PREVCHECK
		}
	}
	if !validPrev {
		log.Printf("VisitFrom: %v is not a valid existing VisitNode", prev)
	}

	g.markVisit(prev, n)
}

// LastNode returns the last node in the VisitGraph.
func (g *VisitGraph) LastNode() *VisitNode {
	if len(g.nodes) == 0 {
		log.Printf("VisitGraph.LastNode: empty graph")
		return nil
	}
	return g.nodes[len(g.nodes)-1]
}

// Size of the graph.
func (g *VisitGraph) Size() int {
	return len(g.nodes)
}

// NodeVisited returns true if the block is visited.
// A block is visited if all the in edges are visited.
func (g *VisitGraph) NodeVisited(toVisit *VisitNode) bool {
	g.Lock()
	defer g.Unlock()
	if toVisit.Blk() == nil {
		log.Println("Block is nil - assume visited = true")
		return true
	}
	if g.nodes == nil { // First visit (main.init#0)
		return false
	}
	allVisited := true
	if fn, ok := g.visited[toVisit.Fn()]; ok { // Fn is visited before
		inEdges, _ := fn[toVisit.Index()]
		for to := range inEdges {
			allVisited = allVisited && inEdges[to]
		}
		return allVisited
	}
	return false
}

// EdgeVisited returns true if the edge between the node pair has been visited.
func (g *VisitGraph) EdgeVisited(from, to *VisitNode) bool {
	g.Lock()
	defer g.Unlock()
	if to.Blk() == nil {
		log.Println("Block is nil - assume visited = true")
		return true
	}
	if g.nodes == nil { // First visit (main.init#0)
		return false
	}
	if fn, ok := g.visited[to.Fn()]; ok { // Fn is visited before
		inEdges, _ := fn[to.Index()]
		return inEdges[from.Index()]
	}
	return false
}

// NodePartialVisited returns false if a block has been Visit()'ed at least once,
// otherwise true.
func (g *VisitGraph) NodePartialVisited(toVisit *VisitNode) bool {
	g.Lock()
	defer g.Unlock()
	if toVisit.Blk() == nil {
		log.Println("Block is nil - assume unvisited = false (nothing to visit)")
		return false
	}
	if g.nodes == nil { // First visit (main.init#0)
		return true // Unvisited by default.
	}
	if fn, ok := g.visited[toVisit.Fn()]; ok { // Fn is visited before
		inEdges, _ := fn[toVisit.Index()]
		if len(inEdges) == 0 { // 0 incoming edges
			return true // Can't tell if blk is visited from incoming. Assume unvisited.
		}
		for to := range inEdges {
			if inEdges[to] { // from at least one of the edges
				return false
			}
		}
		return true
	}
	return true
}

// VisitedOnce returns true if the block is visited at least once.
func (g *VisitGraph) VisitedOnce(toVisit *VisitNode) bool {
	g.Lock()
	defer g.Unlock()
	if toVisit.Blk() == nil {
		log.Println("Block is nil - assume visited = true")
		return true
	}
	if g.nodes == nil { // First visit (main.init#0)
		return false
	}
	if fn, ok := g.visited[toVisit.Fn()]; ok { // Fn is visited before
		inEdges, _ := fn[toVisit.Index()]
		if len(inEdges) == 0 { // 0 incoming edges
			return true
		}
		for to := range inEdges {
			if inEdges[to] { // from at least one of the edges
				return true
			}
		}
	}
	return false
}

// VisitNode is one node in the VisitGraph.
// Each VisitNode corresponds to one ssa.BasicBlock.
type VisitNode struct {
	blk *ssa.BasicBlock // Block.

	Prev, Next *VisitNode
}

// NewVisitNode reutrns a new VisitNode.
func NewVisitNode(block *ssa.BasicBlock) *VisitNode {
	return &VisitNode{blk: block}
}

// Blk returns the underlying BasicBlock.
func (n *VisitNode) Blk() *ssa.BasicBlock {
	return n.blk
}

// Fn returns the function which the block belongs to.
func (n *VisitNode) Fn() *ssa.Function {
	if n.blk == nil {
		log.Fatal(ErrBadNode)
	}
	return n.blk.Parent()
}

// Index returns the block index.
func (n *VisitNode) Index() int {
	if n.blk == nil {
		log.Fatalln(ErrBadNode)
	}
	return n.blk.Index
}

func (n *VisitNode) String() string {
	if n.Next != nil {
		return fmt.Sprintf("Block: %+v#%d\n%s",
			n.Fn(), n.Index(), n.Next.String())
	}
	return fmt.Sprintf("Block: %+v#%d\n-- end --\n", n.Fn(), n.Index())
}
