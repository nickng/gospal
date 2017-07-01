// Package fn provides the Analyser interface for functions and supporting
// utils.
package fn

import "golang.org/x/tools/go/ssa"

// Analyser is an interface for Function analysis,
// handles function entry and exit.
type Analyser interface {
	// EnterFunc analyses a Function.
	EnterFunc(fn *ssa.Function)

	// ExitFunc finishes analysing a Function.
	// It should be used for cleanup etc.
	ExitFunc(fn *ssa.Function)
}
