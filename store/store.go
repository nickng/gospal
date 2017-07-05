// Package store provides interfaces for a key-value store.
// Keys are nameable objects, and Values have a unique representation.
package store

import (
	"bytes"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"log"

	"golang.org/x/tools/go/ssa"
)

// Key is a nameable key for lookup in storage.
//
// Key represents a variable where its name is unique in the scope.
// Key may be synthetic, in such case, the position should be set to NoPos to
// indicate that the Key does not originate from the program.
//
// The string description of the key and the Position is primarily for debugging
// and locating source of the key. The underlying datatype is needed for
// compositional datatypes (struct, chan, map, etc.).
//
type Key interface {
	Name() string     // (Short) name of key.
	Pos() token.Pos   // Position in program.
	String() string   // Description of key.
	Type() types.Type // Underlying datatype.
}

// A Value represents the (symbolic) value in a storage.
//
// The value could be any symbolic/concrete representation of a variable
// instance, but is required to be unique so difference instances can be
// distinguished.
//
type Value interface {
	UniqName() string
}

// Store is a two-layer key-value storage for ssa.Value.
//
// Effectively the storage is a map[Key]map[Value]ssa.Value, but the second
// layer is designed to be efficient for name substitution.
type Store struct {
	logger *log.Logger
	names  map[Key]Value
	vals   *Pool // Actual object storage.
}

func New() *Store {
	return &Store{
		logger: log.New(ioutil.Discard, "store: ", 0),
		names:  make(map[Key]Value),
		vals:   newPool(),
	}
}

// Extend storage with new set of names, using the same backing storage.
func Extend(s *Store) *Store {
	return &Store{
		logger: s.logger,
		names:  make(map[Key]Value),
		vals:   s.vals,
	}
}

// Get retrieves the Value in storage give Key k, if k is not found returns a
// MockValue.
func (s *Store) Get(k Key) Value {
	if v, ok := s.names[k]; ok {
		s.logger.Printf("Get: %s ↦ %v\t%s", k.Name(), v.UniqName(), k.Type())
		return v
	}
	if c, ok := k.(*ssa.Const); ok {
		s.logger.Printf("Get const: %s ↦ %v\t%s", k.Name(), c.String(), k.Type())
		return getConst(c)
	}
	s.logger.Printf("Get: %s ↦ (not found)\t%s", k.Name(), k.Type())
	return MockValue{SrcPos: k.Pos(), Description: "Undefined"}
}

// Put inserts an existing value v with the new key k.
func (s *Store) Put(k Key, v Value) {
	s.names[k] = v
	s.logger.Printf("Put: %s ↦ %v\t%s", k.Name(), v.UniqName(), k.Type())
}

// PutObj puts a new, fresh ssa.Value v to storage with key k.
// The new value can be referenced by a centrally generated unqiue ID.
func (s *Store) PutObj(k Key, v ssa.Value) {
	s.logger.Printf("PutObj: %s ↦ %v\t%s", k.Name(), v.Name(), k.Type())
	uniqID := s.vals.AddValue(v)
	s.names[k] = uniqID
}

// PutUniq inserts an existing wrapped value v with the new key k.
func (s *Store) PutUniq(k Key, v ValueWrapper) error {
	if err := s.vals.AddWrapped(v); err != nil {
		return err
	}
	s.names[k] = v
	s.logger.Printf("Put wrapper: %s ↦ %v\t%s", k.Name(), v.UniqName(), k.Type())
	return nil
}

// GetObj obtains the underlying ssa.Value in the actual object store.
func (s *Store) GetObj(k Key) ssa.Value {
	if v, ok := s.names[k]; ok {
		if val, err := s.vals.Get(v); err != nil {
			return val
		}
	}
	return nil // ERROR key not found
}

func (s *Store) String() string {
	var buf bytes.Buffer
	buf.WriteString("┌─────┄ name: val type ┄──────\n")
	for k, v := range s.names {
		buf.WriteString(fmt.Sprintf("│ %v:\t%v\t%s\n",
			k.Name(), v.UniqName(), k.Type().String()))
	}
	buf.WriteString("└─────────────────────────────\n")
	return buf.String()
}

// Logger is the logging interface for a storage.
type Logger interface {
	SetLog(io.Writer)
}

// SetOutput sets debug output stream to w.
func (s *Store) SetLog(w io.Writer) {
	if w != nil {
		s.logger.SetOutput(w)
	}
}

// Const is a known constant.
//
// It's used to track unique symbols of the same constant value.
type Const struct{ ssa.Const }

// UniqueName of Const is its SSA Value name, e.g. 1:int
func (c Const) UniqName() string {
	return c.String()
}

var constants = make(map[*ssa.Const]Const)

// getConst returns a constant where same values gets the same Const.
func getConst(c *ssa.Const) Const {
	if con, ok := constants[c]; ok {
		return con
	}
	con := Const{Const: *c}
	constants[c] = con
	return con
}
