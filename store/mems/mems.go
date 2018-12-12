// Package mems implements store.Value for shared variable.
package mems

import (
	"fmt"

	"github.com/nickng/gospal/store"
	"golang.org/x/tools/go/ssa"
)

// Mem is a wrapper for a heap-allocated
// (pointer-to-basic type) variable.
type Mem struct {
	ssa.Value

	namespace store.Value
}

// New returns a new mem.
func New(callsite store.Value, val ssa.Value) *Mem {
	return &Mem{
		Value:     val,
		namespace: callsite,
	}
}

func (m *Mem) UniqName() string {
	return fmt.Sprintf("%s.%s_mem.%v", m.namespace.UniqName(), m.Value.Name(), m.Pos())
}
