package migoinfer

import (
	"bytes"
	"fmt"

	"github.com/nickng/gospal/callctx"
	"github.com/nickng/gospal/store"
	"github.com/nickng/migo"
)

// MiGo specific helpers.

// Exported is a holder of visible local names in a function.
type Exported struct {
	names []store.Key
}

// Export puts a local variable k in the set of exported names.
// Only exported names can appear in a MiGo function.
//
// Precondition: e only has unique elements.
func (e *Exported) Export(k store.Key) {
	for _, name := range e.names {
		if name.Name() == k.Name() {
			return
		}
	}
	e.names = append(e.names, k)
}

func (e *Exported) Unexport(k store.Key) {
	for i := 0; i < len(e.names); i++ {
		if e.names[i].Name() == k.Name() {
			e.names = append(e.names[:i], e.names[i+1:]...)
			return
		}
	}
}

// FindExported returns name that points the same value v but using an exported
// name.
func (e *Exported) FindExported(ctx callctx.Context, v store.Value) store.Key {
	if e.names == nil {
		return Unexported{Key: store.MockKey{Description: "Unexported value"}, Value: v}
	}
	for _, name := range e.names {
		if ctx.Get(name) == v {
			return name
		}
	}
	return Unexported{Key: store.MockKey{Description: "Unexported value"}, Value: v}
}

func (e *Exported) String() string {
	var buf bytes.Buffer
	buf.WriteString("┌─ Exported\n")
	for _, n := range e.names {
		buf.WriteString(fmt.Sprintf("│ %s\n", n.Name()))
	}
	buf.WriteString("└─")
	return buf.String()
}

type Unexported struct {
	store.Key
	Value store.Value
}

// migoCall returns a 'call' in MiGo using exported values.
func migoCall(fn string, idx int, exported *Exported) migo.Statement {
	var stmt migo.Statement
	var params []*migo.Parameter

	for _, name := range exported.names {
		params = append(params, &migo.Parameter{Caller: name, Callee: name})
	}
	if idx == 0 {
		stmt = &migo.CallStatement{Name: fn, Params: params}
	} else {
		stmt = &migo.CallStatement{Name: fmt.Sprintf("%s#%d", fn, idx), Params: params}
	}
	return stmt
}
