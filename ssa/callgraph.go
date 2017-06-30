package ssa

import (
	"bufio"
	"fmt"
	"go/token"
	"io"

	"github.com/pkg/errors"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/ssa"
)

// CallGraph is a representation of CallGraph, wrapped with metadata.
type CallGraph struct {
	cg      *callgraph.Graph // Internal cached copy of the callgraph.
	edges   []*cgEdge        // Result of callgraph analysis.
	prog    *ssa.Program     // SSA Program for which the callgraph is built from.
	usedFns []*ssa.Function  // Functions actually used by current Program.
	allFns  []*ssa.Function  // Functions in the current Program (including unused).
}

// AllFunctions return all ssa.Functions defined in the current Program.
func (g *CallGraph) AllFunctions() ([]*ssa.Function, error) {
	// If cached.
	if g.allFns != nil {
		return g.allFns, nil
	}

	visited := make(map[*ssa.Function]bool)
	if err := callgraph.GraphVisitEdges(g.cg, func(edge *callgraph.Edge) error {
		if _, ok := visited[edge.Caller.Func]; !ok {
			visited[edge.Caller.Func] = true
		}
		if _, ok := visited[edge.Callee.Func]; !ok {
			visited[edge.Callee.Func] = true
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "callgraph: failed to visit edges")
	}

	for fn := range visited {
		g.allFns = append(g.allFns, fn)
	}
	return g.allFns, nil
}

// UsedFunctions return a slice of ssa.Function actually used by the current
// Program, rooted at main.init() and main.main().
func (g *CallGraph) UsedFunctions() ([]*ssa.Function, error) {
	// Cached.
	if g.usedFns != nil {
		return g.usedFns, nil
	}

	callTree := make(map[*ssa.Function][]*ssa.Function)
	if err := callgraph.GraphVisitEdges(g.cg, func(edge *callgraph.Edge) error {
		callTree[edge.Caller.Func] = append(callTree[edge.Caller.Func], edge.Callee.Func)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "callgraph: failed to visit edges")
	}

	mains, err := MainPkgs(g.prog, false)
	if err != nil {
		return nil, errors.Wrap(err, "callgraph: failed to find main packages (Check if this this a command?)")
	}

	var fnQueue []*ssa.Function
	for _, main := range mains {
		if main.Func("main") != nil {
			fnQueue = append(fnQueue, main.Func("init"), main.Func("main"))
		}
	}

	visited := make(map[*ssa.Function]bool)
	for len(fnQueue) > 0 {
		headFn := fnQueue[0]
		fnQueue = fnQueue[1:]
		visited[headFn] = true
		for _, fn := range callTree[headFn] {
			if _, ok := visited[fn]; !ok { // If not visited
				fnQueue = append(fnQueue, fn)
			}
			visited[fn] = true
		}
	}

	for fn := range visited {
		g.usedFns = append(g.usedFns, fn)
	}
	return g.usedFns, nil
}

// populateEdges populates a slice of edges in the CallGraph.
func (g *CallGraph) populateEdges(edge *callgraph.Edge) error {
	e := &cgEdge{
		Caller:   edge.Caller.Func,
		Callee:   edge.Callee.Func,
		position: token.Position{Offset: -1},
		edge:     edge,
		fset:     g.prog.Fset,
	}
	g.edges = append(g.edges, e)
	return nil
}

// WriteGraphviz writes callgraph to w in graphviz dot format.
func (g *CallGraph) WriteGraphviz(w io.Writer) error {
	if g.edges == nil {
		if err := callgraph.GraphVisitEdges(g.cg, g.populateEdges); err != nil {
			return err
		}
	}

	bufw := bufio.NewWriter(w)
	bufw.WriteString("digraph callgraph {\n")
	// Instead of using template..
	for _, edge := range g.edges {
		bufw.WriteString(fmt.Sprintf("  %q -> %q\n", edge.Caller, edge.Callee))
	}
	bufw.WriteString("}\n")
	bufw.Flush()
	return nil
}

// cgEdge is a single edge in the callgraph.
//
// Code based on golang.org/x/tools/cmd/callgraph
//
type cgEdge struct {
	Caller *ssa.Function
	Callee *ssa.Function

	edge     *callgraph.Edge
	fset     *token.FileSet
	position token.Position // initialized lazily
}

func (e *cgEdge) pos() *token.Position {
	if e.position.Offset == -1 {
		e.position = e.fset.Position(e.edge.Pos()) // called lazily
	}
	return &e.position
}

func (e *cgEdge) Filename() string { return e.pos().Filename }
func (e *cgEdge) Column() int      { return e.pos().Column }
func (e *cgEdge) Line() int        { return e.pos().Line }
func (e *cgEdge) Offset() int      { return e.pos().Offset }

func (e *cgEdge) Dynamic() string {
	if e.edge.Site != nil && e.edge.Site.Common().StaticCallee() == nil {
		return "dynamic"
	}
	return "static"
}

func (e *cgEdge) Description() string { return e.edge.Description() }

// BuildCallGraph constructs a callgraph from ssa.Info.
// algo is algorithm available in golang.org/x/tools/go/callgraph, which
// includes:
//  - static  static calls only (unsound)
//  - cha     Class Hierarchy Analysis
//  - rta     Rapid Type Analysis
//  - pta     inclusion-based Points-To Analysis
//
func (info *Info) BuildCallGraph(algo string, tests bool) (*CallGraph, error) {
	var cg *callgraph.Graph
	switch algo {
	case "static":
		cg = static.CallGraph(info.Prog)

	case "cha":
		cg = cha.CallGraph(info.Prog)

	case "pta":
		ptrCfg, err := info.PtrAnlysCfg(tests)
		if err != nil {
			return nil, err
		}
		ptrCfg.BuildCallGraph = true
		ptares, err := info.RunPtrAnlys(ptrCfg)
		if err != nil {
			return nil, err // internal error in pointer analysis
		}
		cg = ptares.CallGraph

	case "rta":
		mains, err := MainPkgs(info.Prog, tests)
		if err != nil {
			return nil, err
		}
		var roots []*ssa.Function
		for _, main := range mains {
			roots = append(roots, main.Func("init"), main.Func("main"))
		}
		rtares := rta.Analyze(roots, true)
		cg = rtares.CallGraph
	}

	cg.DeleteSyntheticNodes()

	return &CallGraph{cg: cg, prog: info.Prog}, nil
}
