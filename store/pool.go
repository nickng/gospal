package store

import (
	"fmt"
	"sync"

	"golang.org/x/tools/go/ssa"
)

// ObjUndefError is the error returned if accessing a non-existent object.
type ObjUndefError struct {
	uniqID Value
}

func (e ObjUndefError) Error() string {
	return fmt.Sprintf("object undefined (id: %v)", e.uniqID)
}

// IDClashError is the error returned if the unique ID of two objects clash.
type IDClashError struct {
	ID Value
}

func (e IDClashError) Error() string {
	return fmt.Sprintf("object unique ID clashed (id: %v)", e.ID)
}

// Pool is key-value store of ssa.Value to store instances.
//
// Pool key has no particular significance, except that it has to be unique
// within the Pool. Always use AddValue() to add ssa.Value in the Pool, or use
// AddWrapped() to add a pre-wrapped ssa.Value in the Pool. A new unique key is
// generated for each call (multiple of the same value is allowed as long as
// they are added by different NewValue calls).
type Pool struct {
	pool  map[Value]ssa.Value
	count int

	mu sync.Mutex
}

func newPool() *Pool {
	return &Pool{pool: make(map[Value]ssa.Value)}
}

// poolKey is a type for use as unique key in the Pool.
type poolKey int

func (v poolKey) UniqName() string {
	return fmt.Sprintf("pool_%d", v)
}

func (p *Pool) Get(v Value) (ssa.Value, error) {
	if obj, ok := p.pool[v]; ok {
		return obj, nil
	}
	return nil, ObjUndefError{uniqID: v}
}

func (p *Pool) AddValue(v ssa.Value) Value {
	p.mu.Lock()
	defer p.mu.Unlock()
	newVal := poolKey(p.count + 1)
	p.pool[newVal] = v
	p.count++
	return newVal
}

// ValueWrapper is a Value that wraps an ssa.Value.
// It can be used as both key and value of the pool for convenience.
type ValueWrapper interface {
	ssa.Value // Actual Value.
	Value     // Ensure uniqueness.
}

func (p *Pool) AddWrapped(v ValueWrapper) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pool[v]; !ok {
		p.pool[v] = v
		return nil
	}
	return IDClashError{ID: v}
}
