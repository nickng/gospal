package migoinfer

import (
	"github.com/fatih/color"
	"github.com/nickng/gospal/block"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"golang.org/x/tools/go/ssa"
)

// Function is a visitor for functions.
// It does not deal with the body of the functions, but serves as a location for
// context switching.
type Function struct {
	callctx.Context // Function context. Initially parameters, expands as program evolve.

	Env    *Environment    // Program environment.
	Callee *funcs.Instance // Instance of this function.

	block.Analyser // Function body analyser.
	*Logger
}

// NewFunction creates a new function visitor.
//
// NewFunction takes two parameters to setup the call environment.
//   - Program environment: env
//   - Caller context: ctx
//   - Function definition: def
// They contain the global, and caller local variables respectively.
// In particular, the caller context contains the caller *ssa.Function and
// its corresponding call function.
func NewFunction(call *funcs.Call, env *Environment, ctx callctx.Context) *Function {
	callee := funcs.Instantiate(call)
	f := Function{
		Context: callctx.Switch(ctx, callee),
		Env:     env,
		Callee:  callee,
	}
	b := NewBlock(f.Callee, f.Env, f.Context)
	f.Analyser = b
	return &f
}

func (f *Function) EnterFunc(fn *ssa.Function) {
}

func (f *Function) ExitFunc(fn *ssa.Function) {
}

// SetLogger sets logger for Function and its child block.Analyser.
func (f *Function) SetLogger(l *Logger) {
	f.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.CyanString("func "),
	}
	if ls, ok := f.Analyser.(LogSetter); ok {
		ls.SetLogger(f.Logger)
	}
}
