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
	Callee          *funcs.Instance // Instance of this function.
	callctx.Context                 // Function context.
	Env             *Environment    // Program environment.

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
func NewFunction(call *funcs.Call, ctx callctx.Context, env *Environment) *Function {
	callee := funcs.Instantiate(call)
	f := Function{
		Callee:  callee,
		Context: callctx.Switch(ctx, callee),
		Env:     env,
	}
	b := NewBlock(f.Callee, f.Context, f.Env)
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
