package ssa

import (
	"io"
	"sort"

	"golang.org/x/tools/go/ssa"
)

// members is slice of ssa.Member. Used only for sorting by Pos.
type members []ssa.Member

func (m members) Len() int           { return len(m) }
func (m members) Less(i, j int) bool { return m[i].Pos() < m[j].Pos() }
func (m members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// WriteTo writes Functions used by the Program to w in human readable SSA IR
// instruction format.
func (info *Info) WriteTo(w io.Writer) (int64, error) {
	graph, err := info.BuildCallGraph("rta", false)
	if err != nil {
		return 0, err
	}
	funcs, err := graph.UsedFunctions()
	if err != nil {
		return 0, err
	}
	pkgFuncs := make(map[*ssa.Package]members)
	for _, f := range funcs {
		pkgFuncs[f.Pkg] = append(pkgFuncs[f.Pkg], f)
	}
	var n int64
	for pkg := range pkgFuncs {
		sort.Sort(pkgFuncs[pkg])
		for _, f := range pkgFuncs[pkg] {
			written, err := f.(*ssa.Function).WriteTo(w)
			if err != nil {
				return n, err
			}
			n += written
		}
	}
	return n, nil
}

// WriteAll writes all Functions found in the Program to w in human readable SSA
// IR instruction format.
func (info *Info) WriteAll(w io.Writer) (int64, error) {
	graph, err := info.BuildCallGraph("rta", false)
	if err != nil {
		return 0, err
	}
	funcs, err := graph.AllFunctions()
	if err != nil {
		return 0, err
	}
	pkgFuncs := make(map[*ssa.Package]members)
	for _, f := range funcs {
		pkgFuncs[f.Pkg] = append(pkgFuncs[f.Pkg], f)
	}
	var n int64
	for pkg := range pkgFuncs {
		sort.Sort(pkgFuncs[pkg])
		for _, f := range pkgFuncs[pkg] {
			written, err := f.(*ssa.Function).WriteTo(w)
			if err != nil {
				return n, err
			}
			n += written
		}
	}
	return n, nil
}
