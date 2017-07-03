package block

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"

	gssa "github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/ssa/build"
)

func getTestMainFn(t *testing.T) *ssa.Function {
	prog := `package main
	func main() {  // Block 0
		x := 1
		if x < 2 { // Block 1
			x++
		}
		x = 0      // Block 2
		x++
	}`

	conf := build.FromReader(strings.NewReader(prog))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, _ := gssa.MainPkgs(info.Prog, false)
	return mains[0].Func("main")
}

// Tests basic usage of VisitGraph.
func TestVisitGraph(t *testing.T) {
	mainFn := getTestMainFn(t)
	g := NewVisitGraph(false)
	if g.Size() != 0 {
		t.Errorf("New VisitGraph should have 0 node, got %d", g.Size())
	}
	n0 := NewVisitNode(mainFn.Blocks[0])
	g.Visit(n0)
	if g.Size() != 1 {
		t.Errorf("Add node(0) to VisitGraph should make size 1, but got %d",
			g.Size())
	}
	if g.nodes[0] != n0 {
		t.Errorf("nodes[0] should be %+v, got %+v", n0, g.nodes[0])
	}
	n1 := NewVisitNode(mainFn.Blocks[1])
	g.Visit(n1)
	if g.nodes[0] != n0 {
		t.Errorf("Add node(1) to VisitGraph should not change existing root, but got %+v",
			g.nodes[0])
	}
	if g.Size() != 2 {
		t.Errorf("Add node(1) to VisitGraph should make size 2, but got %d", g.Size())
	}
	if g.nodes[1] != n1 {
		t.Errorf("nodes[1] should be %+v, got %+v", n1, g.nodes[1])
	}
}

// Tests that Add works properly.
func TestVisitGraphAdd(t *testing.T) {
	mainFn := getTestMainFn(t)
	g := NewVisitGraph(false)
	n0 := NewVisitNode(mainFn.Blocks[0])
	g.Visit(n0)
	if g.Size() != 1 {
		t.Errorf("Add node(0) to VisitGraph should make it size 1, but got %d",
			g.Size())
	}
	if g.nodes[0] != n0 {
		t.Errorf("nodes[0] should be %+v, got %+v", n0, g.nodes[0])
	}
	n1 := NewVisitNode(mainFn.Blocks[1])
	g.Visit(n1)
	if g.nodes[0] != n0 {
		t.Errorf("Add node(1) to VisitGraph should not change existing root, but got %+v",
			g.nodes[0])
	}
	if g.Size() != 2 {
		t.Errorf("Add node(1) to VisitGraph should make size 2, but got %d", g.Size())
	}
	if g.nodes[1] != n1 {
		t.Errorf("nodes[1] should be %+v, got %+v", n1, g.nodes[1])
	}
	// Make sure the pointers are correct.
	if n0.Prev != nil {
		t.Errorf("nodes[0].Prev should not be pointing to anything, got %+v",
			n0.Prev)
	}
	if n0.Next != n1 {
		t.Errorf("nodes[0].Next should be pointing to %+v, got %+v",
			n1, n0.Next)
	}
	if n1.Prev != n0 {
		t.Errorf("nodes[1].Prev should be pointing to %+v, got %+v",
			n0, n1.Prev)
	}
	if n1.Next != nil {
		t.Errorf("nodes[1].Next should not be pointing to anything, got %+v",
			n1.Next)
	}
	n2 := NewVisitNode(mainFn.Blocks[2])
	g.Visit(n2)
	if g.nodes[2] != n2 {
		t.Errorf("nodes[2] should be %+v, got %+v", n2, g.nodes[2])
	}
	if g.nodes[2].Prev != g.nodes[1] {
		t.Errorf("node[2].Prev should be %+v, got %+v",
			g.nodes[1], g.nodes[2].Prev)
	}
	if g.nodes[1].Next != g.nodes[2] {
		t.Errorf("node[1].Next should be %+v, got %+v",
			g.nodes[2], g.nodes[1].Next)
	}
	if g.nodes[2].Next != nil {
		t.Errorf("node[2].Next should not be pointing to anything, got %+v",
			g.nodes[2].Next)
	}
}

func TestVisitGraphVisited(t *testing.T) {
	mainFn := getTestMainFn(t)
	// According to the control flow, 0 --> { 1 --> 2, 2 }
	g := NewVisitGraph(false)
	b0 := NewVisitNode(mainFn.Blocks[0])
	b1 := NewVisitNode(mainFn.Blocks[1])
	b2 := NewVisitNode(mainFn.Blocks[2])
	if g.Visited(b0) {
		t.Errorf("Block 0 and parent function should be unvisited, got %t",
			g.Visited(b0))
	}
	g.Visit(b0)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if g.Visited(b1) {
		t.Errorf("Block 1 should be unvisited by default, got %t", g.Visited(b1))
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited by default, got %t", g.Visited(b2))
	}

	g.Visit(b1) // If then (if 1 else 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][0])
	}
	if g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][1])
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited, got %t", g.Visited(b2))
	}

	g.Visit(b2) // Follow up on If then (jump 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][1])
	}
	if !g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is visited, got %t", g.visited[b2.Fn()][2][1])
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited, got %t", g.Visited(b2))
	}

	// Revert to if-parent, b0
	g.VisitFrom(b0, b2) // If else (if 1 else 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if !g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is visited, got %t", g.visited[b2.Fn()][2][0])
	}
	if !g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is visited, got %t", g.visited[b2.Fn()][2][1])
	}
	if !g.Visited(b2) {
		t.Errorf("Block 2 should be visited, got %t", g.Visited(b2))
	}
}

func TestVisitGraphVisitedOnce(t *testing.T) {
	mainFn := getTestMainFn(t)
	// According to the control flow, 0 --> { 1 --> 2, 2 }
	g := NewVisitGraph(false)
	b0 := NewVisitNode(mainFn.Blocks[0])
	b1 := NewVisitNode(mainFn.Blocks[1])
	b2 := NewVisitNode(mainFn.Blocks[2])
	if g.Visited(b0) {
		t.Errorf("Block 0 and parent function should be unvisited, got %t",
			g.Visited(b0))
	}
	g.Visit(b0)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.VisitedOnce(b0) {
		t.Errorf("Block 0 should be visited once, got %t", g.VisitedOnce(b0))
	}
	if g.Visited(b1) {
		t.Errorf("Block 1 should be unvisited by default, got %t", g.Visited(b1))
	}
	if g.VisitedOnce(b1) {
		t.Errorf("Block 1 should be unvisited once by default, got %t", g.VisitedOnce(b1))
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited by default, got %t", g.Visited(b2))
	}
	if g.VisitedOnce(b1) {
		t.Errorf("Block 1 should be unvisited once by default, got %t", g.VisitedOnce(b1))
	}

	g.Visit(b1) // If then (if 1 else 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.VisitedOnce(b0) {
		t.Errorf("Block 0 should be visited once, got %t", g.VisitedOnce(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if !g.VisitedOnce(b1) {
		t.Errorf("Block 1 should be visited once, got %t", g.VisitedOnce(b1))
	}
	if g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][0])
	}
	if g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][1])
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited, got %t", g.Visited(b2))
	}
	if g.VisitedOnce(b2) {
		t.Errorf("Block 2 should be unvisited, got %t", g.VisitedOnce(b2))
	}

	g.Visit(b2) // Follow up on If then (jump 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.VisitedOnce(b0) {
		t.Errorf("Block 0 should be visited once, got %t", g.VisitedOnce(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if !g.VisitedOnce(b1) {
		t.Errorf("Block 1 should be visited once, got %t", g.VisitedOnce(b1))
	}
	if g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is unvisited, got %t", g.visited[b2.Fn()][2][1])
	}
	if !g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is visited, got %t", g.visited[b2.Fn()][2][1])
	}
	if g.Visited(b2) {
		t.Errorf("Block 2 should be unvisited, got %t", g.Visited(b2))
	}
	if !g.VisitedOnce(b2) {
		t.Errorf("Block 2 should be visited once (of two), got %t", g.VisitedOnce(b2))
	}

	// Revert to if-parent, b0
	g.VisitFrom(b0, b2) // If else (if 1 else 2)
	if !g.Visited(b0) {
		t.Errorf("Block 0 should be visited, got %t", g.Visited(b0))
	}
	if !g.VisitedOnce(b0) {
		t.Errorf("Block 0 should be visited once, got %t", g.VisitedOnce(b0))
	}
	if !g.Visited(b1) {
		t.Errorf("Block 1 should be visited, got %t", g.Visited(b1))
	}
	if !g.VisitedOnce(b1) {
		t.Errorf("Block 1 should be visited once, got %t", g.VisitedOnce(b1))
	}
	if !g.visited[b2.Fn()][2][0] {
		t.Errorf("0 --> 2 is visited, got %t", g.visited[b2.Fn()][2][0])
	}
	if !g.visited[b2.Fn()][2][1] {
		t.Errorf("1 --> 2 is visited, got %t", g.visited[b2.Fn()][2][1])
	}
	if !g.Visited(b2) {
		t.Errorf("Block 2 should be visited, got %t", g.Visited(b2))
	}
	if !g.VisitedOnce(b2) {
		t.Errorf("Block 2 should be visited once, got %t", g.VisitedOnce(b2))
	}
}

// Tests reentrant block graph.
func TestVisitGraphReentrant(t *testing.T) {
	mainFn := getTestMainFn(t)
	g := NewVisitGraph(true)
	b0 := NewVisitNode(mainFn.Blocks[0])
	b1 := NewVisitNode(mainFn.Blocks[1])
	b2 := NewVisitNode(mainFn.Blocks[2])
	g.Visit(b0)
	g.VisitFrom(b0, b1) // Then branch (toplevel).
	if !g.Visited(b1) {
		t.Errorf("0 --> 1 should mark 1 visited, got %t", g.Visited(b1))
	}
	t.Logf("Before call: %+v", g.visited)
	g.Visit(b0) // main calls main in (enters level 1)
	t.Logf("After call: %+v", g.visited)
	if g.Visited(b1) {
		t.Errorf("0 --> 1 --> (0 reentrant) should have 1 unvisited, got %t",
			g.Visited(b1))
	}
	g.VisitFrom(b0, b2)
	g.MarkLast(b2) // exits level 1, pop stack.
	t.Logf("After return: %+v", g.visited)
	if !g.Visited(b1) {
		t.Errorf("0 --> 1 --> (0 reentrant --> 2 end) should have 1 visited, got %t",
			g.Visited(b1))
	}
	g.Visit(b2)
	g.VisitFrom(b0, b2) // Else branch (toplevel).
	if g.Visited(b2) {
		t.Errorf("0 --> { 1 --> (0 reentrant --> 2 end) --> 2, 2 } should have 2 visited, got %t",
			g.Visited(b2))
	}
}

// Tests non-reentrant block graph.
func TestVisitGraphReentrantFalse(t *testing.T) {
	mainFn := getTestMainFn(t)
	g := NewVisitGraph(false)
	b0 := NewVisitNode(mainFn.Blocks[0])
	b1 := NewVisitNode(mainFn.Blocks[1])
	b2 := NewVisitNode(mainFn.Blocks[2])
	g.Visit(b0)
	g.VisitFrom(b0, b1) // Then branch (toplevel).
	if !g.Visited(b1) {
		t.Errorf("0 --> 1 should mark 1 visited, got %t", g.Visited(b1))
	}
	t.Logf("Before call: %+v", g.visited)
	g.Visit(b0) // main calls main in (enters level 1), resets blockgraph.
	t.Logf("After call: %+v", g.visited)
	if !g.Visited(b1) {
		t.Errorf("0 --> 1 --> (0) should have 1 visited (unchanged from toplevel), got %t",
			g.Visited(b1))
	}
	g.VisitFrom(b0, b2)
	g.MarkLast(b2) // exits level 1, but since it is non-reentrant, it is noop.
	t.Logf("After return: %+v", g.visited)
	if !g.Visited(b1) {
		t.Errorf("0 --> 1 --> (0 --> 2 end) should have 1 visited, got %t",
			g.Visited(b1))
	}
	if g.Visited(b2) {
		t.Errorf("0 --> { 1 (0 --> 2 end), 2 } should have 2 visited, got %t",
			g.Visited(b2))
	}
	g.Visit(b2)
	g.VisitFrom(b0, b2) // Else branch (toplevel).
	if g.Visited(b2) {
		t.Errorf("0 --> { 1 --> (0 --> 2 end) --> 2, 2 } should have 2 visited, got %t",
			g.Visited(b2))
	}
}
