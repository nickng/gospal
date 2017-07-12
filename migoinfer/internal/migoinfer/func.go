package migoinfer

import (
	"github.com/fatih/color"
	"github.com/nickng/gospal/block"
	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/pkg/errors"
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
	*Exported
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
		Callee:   callee,
		Context:  callctx.Switch(ctx, callee),
		Env:      env,
		Exported: new(Exported),
	}
	for _, param := range f.Callee.Definition().Parameters {
		if isChan(param) {
			f.Export(param)
		}
	}
	b := NewBlock(f.Callee, f.Context, f.Env)
	if b != nil {
		b.Exported = f.Exported
	}
	f.Analyser = b
	return &f
}

// EnterFunc enters a function and perform a context switch.
// This should be the entry point of a function call.
func (f *Function) EnterFunc(fn *ssa.Function) {
	if fn == nil {
		f.Env.Errors <- errors.Wrap(ErrFnIsNil, "When entering function")
	}
	defer f.ExitFunc(fn)
	nBlock := len(f.Callee.Function().Blocks)
	f.Logger.Debugf("%s Enter %s (%d blocks)", f.Logger.Module(), fn.Name(), nBlock)

	if nBlock > 0 {
		// This will visit all blocks in the function.
		f.EnterBlk(f.Callee.Function().Blocks[0])
	}
}

// ExitFunc finalises analysis of a function.
func (f *Function) ExitFunc(fn *ssa.Function) {
	if fn != nil {
		f.Logger.Debugf("%s Exit %s", f.Logger.Module(), fn.Name())
	}
	if b, ok := f.Analyser.(*Block); b != nil && ok {
		// Since a function is complete analysed, we can print its content.
		for _, data := range b.meta {
			f.Env.Prog.AddFunction(data.migoFunc)
		}
	}
}

// SetLogger sets logger for Function and its child block.Analyser.
func (f *Function) SetLogger(l *Logger) {
	f.Logger = &Logger{
		SugaredLogger: l.SugaredLogger,
		module:        color.CyanString("func "),
	}
	if b, ok := f.Analyser.(*Block); b != nil && ok {
		if ls, ok := f.Analyser.(LogSetter); f.Analyser != nil && ok {
			ls.SetLogger(f.Logger)
		}
	}
}
