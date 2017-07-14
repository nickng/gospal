package ssa

import (
	"regexp"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// FindFunc parses path (e.g. "github.com/nickng/gospal/ssa".MainPkgs) and
// returns Function body in SSA IR.
func (info *Info) FindFunc(path string) (*ssa.Function, error) {
	pkgPath, fnName := parseFuncPath(path)
	graph, err := info.BuildCallGraph("rta", false)
	if err != nil {
		return nil, err
	}
	funcs, err := graph.UsedFunctions()
	if err != nil {
		return nil, err
	}
	for _, f := range funcs {
		if f.Pkg.Pkg.Path() == pkgPath && f.Name() == fnName {
			return f, nil
		}
	}
	return nil, nil
}

// parseFuncPath splits path to package and function segments.
// Does not handle complex functions with receivers.
func parseFuncPath(path string) (pkgPath, fnName string) {
	if len(path) < 1 {
		return "", ""
	}
	switch path[0] {
	case '(':
		regex := regexp.MustCompile(`\((?P<pkg>[^)]+)\).(?P<fn>.+)`)
		submatches := regex.FindStringSubmatch(path)
		if len(submatches) >= 3 {
			return submatches[1], submatches[2]
		}
	case '"':
		regex := regexp.MustCompile(`"(?P<pkg>[^)]+)".(?P<fn>.+)`)
		submatches := regex.FindStringSubmatch(path)
		if len(submatches) >= 3 {
			return submatches[1], submatches[2]
		}
	default:
		parts := strings.Split(path, ".")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}
	return "", path
}
