package migoinfer

import (
	"go/token"
	"log"
	"os"

	gssa "github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/store"
	"github.com/nickng/migo"
	"golang.org/x/tools/go/ssa"
)

// Environment captures the global environment of the program shared across
// functions.
type Environment struct {
	Prog        *migo.Program
	Info        *gssa.Info
	Globals     *store.Store
	Errors      chan error
	SkipPkg     map[*ssa.Package]bool
	VisitedFunc map[*ssa.CallCommon]bool
}

// NewEnvironment initialises a new environment.
func NewEnvironment(info *gssa.Info) Environment {
	return Environment{
		Prog:        migo.NewProgram(),
		Info:        info,
		Globals:     store.New(),
		Errors:      make(chan error),
		VisitedFunc: make(map[*ssa.CallCommon]bool),
	}
}

type Poser interface {
	Pos() token.Pos
}

func (env Environment) HandleErrors() {
	logger := log.New(os.Stderr, "ERROR: ", 0)
	for err := range env.Errors {
		if p, ok := err.(Poser); ok {
			logger.Printf("%s: %s", env.Info.FSet.Position(p.Pos()).String(), err)
		} else {
			logger.Println(err)
		}
	}
}

// getPos returns a string representation of the given item.
// Note this is a pointer receiver on Environment for use by the Visitors.
func (env *Environment) getPos(p Poser) string {
	return env.Info.FSet.Position(p.Pos()).String()
}
