package funcs

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"log"

	"github.com/nickng/gospal/store"

	"golang.org/x/tools/go/ssa"
)

// returnCount counts number of occurrences of return value ssa.Value.
type returnCount map[store.Key]int

// returnCounts counts number of occurrences of all return values.
type returnCounts []returnCount

// Definition is a uniform representation for a function for abstracting
// function-like constructs in Go, namely:
//
//  - Builtin function
//  - Ordinary function
//  - Pointer to function (function in a variable)
//  - Closures (functions that carries variable with it)
//
// In this form, a function definition consolidates all parameters to a function
// including captured free variables (closure), and return values to parameters.
// Structures are flattened so that all fields are represented separately.
//
type Definition struct {
	Function   *ssa.Function // Function parameters, returns, signature and body.
	Parameters []store.Key   // Aggregated parameters.
	bindings   []ssa.Value   // Variable bindings.
	returnSet  returnCounts  // Return value from function body.
	NParam     int           // Number of formal parameters.
	NFreeVar   int           // Number of captures.
	NReturn    int           // Number of return values.
	IsVararg   bool          // Is the last parameter variadic?
}

// MakeClosureDefinition returns a definition for a given closure function +
// bindings.
func MakeClosureDefinition(fn *ssa.Function, bindings []ssa.Value) *Definition {
	d := MakeDefinition(fn)
	d.bindings = bindings
	return d
}

// MakeDefinition returns a definition for a given function.
func MakeDefinition(fn *ssa.Function) *Definition {
	if fn == nil {
		log.Fatal("funcs.MakeDefinition: Function is nil")
		return nil
	}
	params, isVararg := getParams(fn)
	nParam := len(params)
	freevars := getFreeVars(fn)
	nFreeVar := len(freevars)
	returns := getReturns(fn)
	nReturn := len(returns)
	def := Definition{
		Function:   fn,
		Parameters: make([]store.Key, nParam+nFreeVar+nReturn),
		NParam:     nParam,
		NFreeVar:   nFreeVar,
		NReturn:    nReturn,
		IsVararg:   isVararg,
	}
	for i := 0; i < nParam; i++ {
		def.Parameters[i] = params[i]
	}
	for i := 0; i < nFreeVar; i++ {
		def.Parameters[nParam+i] = freevars[i]
	}
	// def.Parameters[i | nFreeVar <= i < nFreeVar+nReturn] = nil
	def.returnSet = returns
	for i := 0; i < nReturn; i++ {
		def.Parameters[nParam+nFreeVar+i] = getCommonRetval(returns[i])
	}
	return &def
}

// PAram returns the i-th function parameter of the function definition.
func (d *Definition) Param(i int) store.Key {
	return d.Parameters[i]
}

// FreeVar returns the i-th free variable of the function definition.
func (d *Definition) FreeVar(i int) store.Key {
	return d.Parameters[d.NParam+i]
}

// Return returns the i-th return values in the function body.
// Note that it only returns the most commonly used
func (d *Definition) Return(i int) store.Key {
	return d.Parameters[d.NParam+d.NFreeVar+i]
}

// IsReturn return true if the given name is a return value.
func (d *Definition) IsReturn(k store.Key) bool {
	for _, rc := range d.returnSet {
		for r := range rc {
			if r.Name() == k.Name() {
				return true
			}
		}
	}
	return false
}

// getName returns the function name as "package".function_name.
func (d *Definition) getName() []byte {
	var buf bytes.Buffer
	if r := d.Function.Signature.Recv(); r != nil {
		buf.WriteString(fmt.Sprintf("\"%s\".%s", r.Pkg().Path(), r.Name()))
	} else {
		if pkg := d.Function.Package(); pkg != nil {
			buf.WriteString(fmt.Sprintf("\"%s\"", pkg.Pkg.Path()))
		}
	}
	buf.WriteString(fmt.Sprintf(".%s", d.Function.Name()))
	return buf.Bytes()
}

func (d *Definition) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("func def(%d): ", len(d.Parameters)))
	buf.Write(d.getName())
	buf.WriteRune(' ')

	for i := 0; i < d.NParam; i++ { // Parameters.
		if i > 0 {
			buf.WriteString(", ")
		}
		p := d.Param(i)
		buf.WriteString(fmt.Sprintf("%s:%v", p.Name(), p.Type()))
	}
	if d.IsVararg {
		buf.WriteRune('…')
	}
	for i := 0; i < d.NFreeVar; i++ { // Free variables.
		if i > 0 || d.NParam > 0 {
			buf.WriteString(", ")
		}
		fv := d.FreeVar(i)
		buf.WriteString(fmt.Sprintf("🆓%s:%v", fv.Name(), fv.Type()))
	}
	if d.NReturn > 0 {
		buf.WriteString(" ⇒ {")
		for i := 0; i < d.NReturn; i++ {
			if i > 0 {
				buf.WriteString(", ")
			}
			retval := d.Return(i)
			buf.WriteString(fmt.Sprintf("●:%v", retval.Type()))
		}
		buf.WriteRune('}')
	}
	return buf.String()
}

func (d *Definition) UniqName() string {
	return string(d.getName())
}

// hasBody returns true if the function has body defined.
func hasBody(fn *ssa.Function) bool {
	return len(fn.Blocks) > 0
}

// hasParams returns true if the function has parameters defined.
func hasParams(sig *types.Signature) bool {
	return sig.Recv() != nil || sig.Params() != nil
}

// getParams returns the list of parameters and if the parameter is vararg.
func getParams(fn *ssa.Function) ([]store.Key, bool) {
	if hasBody(fn) {
		nParam := len(fn.Params)
		params := make([]store.Key, nParam)
		for i, param := range fn.Params {
			params[i] = param
		}
		return params, fn.Signature.Variadic()
	}
	// Not concrete but has valid function signature.
	if hasParams(fn.Signature) {
		var params []store.Key
		if sigRecv := fn.Signature.Recv(); sigRecv != nil {
			params = append(params, createMock(fn, sigRecv.Type(), "recv"))
		}
		if sigParams := fn.Signature.Params(); sigParams != nil {
			for i := 0; i < sigParams.Len(); i++ {
				params = append(params, createMock(fn, sigParams.At(i).Type(), "param"))
			}
		}
		return params, fn.Signature.Variadic()
	}
	return nil, false
}

// getFreeVars returns free variables for a function.
func getFreeVars(fn *ssa.Function) []store.Key {
	nFreeVar := len(fn.FreeVars)
	freevars := make([]store.Key, nFreeVar)
	for i, freevar := range fn.FreeVars {
		freevars[i] = freevar
	}
	return freevars
}

// getReturns returns versions of return values, and the tuple size.
func getReturns(fn *ssa.Function) returnCounts {
	var nRetval int
	if retval := fn.Signature.Results(); retval != nil {
		nRetval = retval.Len()
	}
	if hasBody(fn) {
		returns := make([]returnCount, nRetval)
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				switch instr := instr.(type) {
				case *ssa.Return:
					for i, result := range instr.Results {
						if returns[i] == nil {
							returns[i] = make(returnCount)
						}
						returns[i][get(result)]++
					}
				}
			}
		}
		return returns
	}
	if retval := fn.Signature.Results(); retval != nil {
		retvals := make([]returnCount, nRetval)
		for i := 0; i < nRetval; i++ {
			retvals[i] = make(returnCount)
			retvals[i][createMock(fn, retval.At(i).Type(), "retval")] = 1
		}
		return retvals
	}
	return nil
}

// getCommonRetval return the most frequently used return value if multiple
// return value is used for the same tuple-index.
func getCommonRetval(rc returnCount) store.Key {
	if len(rc) == 1 {
		for r := range rc {
			return r
		}
	}
	var max int
	var retval store.Key
	for r, count := range rc { // Find max retval.
		if count > max {
			max = count
			retval = r
		}
	}
	return retval
}

func get(v ssa.Value) ssa.Value {
	switch v := v.(type) {
	case *ssa.ChangeType:
		return get(v.X)
	case *ssa.UnOp:
		if v.Op == token.MUL {
			return get(v.X)
		}
	}
	return v
}
