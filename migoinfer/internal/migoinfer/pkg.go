package migoinfer

import (
	"github.com/fatih/color"
	"golang.org/x/tools/go/ssa"
)

// Package is a visitor for package variables and initialisation.
// None of the data are stored in the visitor (global variables are in shared
// environment) so this can be reused for all packages.
type Package struct {
	env *Environment // Program environment
	*Logger
}

func NewPackage(env *Environment) *Package {
	return &Package{env: env}
}

// InitGlobals initialises package-global varables in environment.
func (p *Package) InitGlobals(*ssa.Package) {
}

// VisitInit visits init function(s) in the package with a fresh context.
func (p *Package) VisitInit(*ssa.Package) {
}

// SetLogger sets logger for Package.
func (p *Package) SetLogger(l *Logger) {
	p.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.BlueString("pkg  "),
	}
}
