package ssa

import (
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// MainPkgs returns the main packages in the program.
func MainPkgs(prog *ssa.Program, tests bool) ([]*ssa.Package, error) {
	pkgs := prog.AllPackages()

	var mains []*ssa.Package
	if tests {
		for _, pkg := range pkgs {
			if main := prog.CreateTestMainPackage(pkg); main != nil {
				mains = append(mains, main)
			}
		}
		if mains == nil {
			return nil, ErrNoTestMainPkgs
		}
		return mains, nil
	}

	mains = append(mains, ssautil.MainPackages(pkgs)...)
	if len(mains) == 0 {
		return nil, ErrNoMainPkgs
	}
	return mains, nil
}
