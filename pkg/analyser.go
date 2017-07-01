// Package pkg provides the Analyser interface for package.
package pkg

import "golang.org/x/tools/go/ssa"

// Analyser is an interface for Package analysis,
// handles package-level variables (globals) and init functions.
type Analyser interface {
	InitGlobals(*ssa.Package)
	VisitInit(*ssa.Package)
}
