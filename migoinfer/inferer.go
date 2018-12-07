package migoinfer

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/funcs"
	"github.com/nickng/gospal/migoinfer/internal/migoinfer"
	"github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/store"
	"github.com/nickng/migo"
	"github.com/nickng/migo/transform"
)

// Inferer is the main MiGo inference entry point.
type Inferer struct {
	Env       migoinfer.Environment // Program environment.
	Info      *ssa.Info             // SSA IR.
	MiGo      *migo.Program         // MiGo program.
	EntryFunc string

	Raw bool

	outWriter io.Writer // Output stream.
	errWriter io.Writer // Error stream.
	*migoinfer.Logger
}

// New returns a new Inferer, and uses w for logging messages.
func New(info *ssa.Info, w io.Writer) *Inferer {
	inferer := Inferer{
		Env:       migoinfer.NewEnvironment(info),
		Info:      info,
		MiGo:      migo.NewProgram(),
		Raw:       false,
		outWriter: ioutil.Discard,
		errWriter: ioutil.Discard,
		Logger:    newLogger(),
	}
	if w != nil {
		inferer.errWriter = w
	}
	return &inferer
}

func (i *Inferer) SetEntryFunc(path string) {
	i.EntryFunc = path
}

func (i *Inferer) Analyse() {
	go i.Env.HandleErrors()
	// Sync error ignored. See https://github.com/uber-go/zap/issues/328
	defer i.Logger.Sync()

	pkg := migoinfer.NewPackage(&i.Env)
	pkg.SetLogger(i.Logger)
	// Package/global variables initialisation.
	for _, p := range i.Info.Prog.AllPackages() {
		pkg.InitGlobals(p)
		pkg.VisitInit(p)
	}
	if i.EntryFunc == "" { // main.main
		// Find main packages to start analysis.
		mains, err := ssa.MainPkgs(i.Info.Prog, false)
		if err != nil {
			log.Fatal("Cannot find main package:", err)
		}
		// Call context
		ctx := callctx.Toplevel()
		if l, ok := ctx.(store.Logger); ok {
			l.SetLog(i.errWriter)
		}
		for _, main := range mains {
			if mainFn := main.Func("main"); mainFn != nil {
				mainDef := funcs.MakeCall(funcs.MakeDefinition(mainFn), nil, nil)
				mainFnAnalyser := migoinfer.NewFunction(mainDef, ctx, &i.Env)
				mainFnAnalyser.SetLogger(i.Logger)
				mainFnAnalyser.EnterFunc(mainDef.Function())
			}
		}
	} else {
		fn, err := i.Info.FindFunc(i.EntryFunc)
		if err != nil {
			log.Fatalf("Cannot find entry function %s", i.EntryFunc)
		}
		ctx := callctx.Toplevel()
		if l, ok := ctx.(store.Logger); ok {
			l.SetLog(i.errWriter)
		}
		if fn != nil {
			fnDef := funcs.MakeCall(funcs.MakeDefinition(fn), nil, nil)
			fnAnalyser := migoinfer.NewFunction(fnDef, ctx, &i.Env)
			fnAnalyser.SetLogger(i.Logger)
			fnAnalyser.EnterFunc(fnDef.Function())
		}
	}
	// Remove unrelated migo functions.
	if !i.Raw {
		for _, f := range i.Env.Prog.Funcs {
			if i.EntryFunc == "" && f.SimpleName() == "main.main" {
				transform.RemoveTauFuncs(i.Env.Prog, f)
				transform.RemoveUndefined(i.Env.Prog)
				break
			} else if f.SimpleName() == i.EntryFunc {
				transform.RemoveTauFuncs(i.Env.Prog, f)
				transform.RemoveUndefined(i.Env.Prog)
				break
			}
		}
	}
	if i.EntryFunc == "" { // main.main
		// Print main.main first.
		for _, f := range i.Env.Prog.Funcs {
			if f.SimpleName() == "main.main" {
				fmt.Fprintf(i.outWriter, f.String())
			}
		}
		for _, f := range i.Env.Prog.Funcs {
			if f.SimpleName() != "main.main" {
				fmt.Fprintf(i.outWriter, f.String())
			}
		}
	}
}

// AddLogFiles extends current Logger and writes additional log to files.
func (i *Inferer) AddLogFiles(file ...string) {
	i.Logger = newFileLogger(file...)
}

func (i *Inferer) SetOutput(w io.Writer) {
	if w != nil {
		i.outWriter = w
	}
}
