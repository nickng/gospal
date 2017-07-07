package migoinfer

import (
	"github.com/fatih/color"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"golang.org/x/tools/go/ssa"
)

// Package is a visitor for package variables and initialisation.
// None of the data are stored in the visitor (global variables are in shared
// environment) so this can be reused for all packages.
type Package struct {
	Env *Environment // Program environment
	*Logger
}

func NewPackage(env *Environment) *Package {
	return &Package{Env: env}
}

// InitGlobals initialises package-global varables in environment.
func (p *Package) InitGlobals(pkg *ssa.Package) {
	for name, memb := range pkg.Members {
		p.Logger.Debugf("%s Package member \"%s\".%s\t%T",
			p.Logger.Module(), pkg.Pkg.Path(), name, memb)
		switch value := memb.(type) {
		case *ssa.Global:
			p.Env.Globals.PutObj(value, value)
			// TODO(nickng) handle special value kinds (array/slice/struct)
		}
	}
}

// VisitInit visits init function(s) in the package with a fresh context.
func (p *Package) VisitInit(pkg *ssa.Package) {
	if initFn := pkg.Func("init"); initFn != nil {
		initDef := funcs.MakeCall(funcs.MakeDefinition(initFn), nil, nil)
		fn := NewFunction(initDef, callctx.Toplevel(), p.Env)
		fn.SetLogger(p.Logger)
		fn.EnterFunc(initDef.Function())
		return
	}
	p.Logger.Warnf("%s %s has no init", p.Logger.Module(), pkg.String())
}

// SetLogger sets logger for Package.
func (p *Package) SetLogger(l *Logger) {
	p.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.BlueString("pkg  "),
	}
}
