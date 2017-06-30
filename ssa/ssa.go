// Package ssa is a library to build and work with SSA.
// For most part the package contains helper or wrapper functions to use the
// packages in Go project's extra tools.
//
// In particular, the SSA IR is from golang.org/x/tools/go/ssa, and reuses many
// of the packages in the static analysis stack built on top of it.
//
package ssa

import (
	"go/token"
	"io"
	"log"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

// Info holds the results of a SSA build for analysis.
// To populate this structure, the 'build' subpackage should be used.
//
type Info struct {
	IgnoredPkgs []string // Record of ignored package during the build process.

	FSet  *token.FileSet  // FileSet for parsed source files.
	Prog  *ssa.Program    // SSA IR for whole program.
	LProg *loader.Program // Loaded program from go/loader.

	BldLog io.Writer // Build log.
	PtaLog io.Writer // Pointer analysis log.

	Logger *log.Logger // Build logger.
}
