// Package chans implements store.Value of type chan.
package chans

import (
	"fmt"

	"github.com/nickng/gospal/store"
	"golang.org/x/tools/go/ssa"
)

// Chan is a wrapper for a type chan SSA value.
type Chan struct {
	ssa.Value
	size int64

	ns store.Value // Namespace.
}

func New(callsite store.Value, ch ssa.Value, size int64) *Chan {
	return &Chan{
		ns:    callsite,
		Value: ch,
		size:  size,
	}
}

func (c *Chan) Size() int64 {
	return c.size
}

func (c *Chan) UniqName() string {
	return fmt.Sprintf("%s.%s_chan%d", c.ns.UniqName(), c.Value.Name(), c.size)
}
