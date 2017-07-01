// Package prog provides the Analyser interface for programs.
package prog

// Analyser is an interface for Program analysis.
type Analyser interface {
	// Analyse is the entry point to the static analyser.
	Analyse()
}
