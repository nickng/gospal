// Package structs implements store.Value for composite struct type.
package structs

import (
	"fmt"
	"go/token"
	"go/types"
	"log"

	"github.com/nickng/gospal/store"
	"golang.org/x/tools/go/ssa"
)

// Struct is a wrapper for a type struct SSA value.
//
// Struct keeps track of the fields (and their respective instances).
// When used as a storage.Key, Struct is the handle to the struct.
// When used as a storage.Value, Struct is a Field of the holding Struct,
// but itself is a sub-Struct.
//
// Do not use a Struct across scope. Make a copy so that the fields do not get
// overwritten.
type Struct struct {
	ssa.Value

	ns     store.Value // Namespace.
	Fields []Field     // Fields references.
}

// Type is an interface for creating new struct from typable things.
type Typer interface {
	Type() types.Type
}

func New(scope store.Value, v Typer) *Struct {
	var s *Struct
	switch v := v.Type().Underlying().(type) {
	case *types.Struct:
		s = FromType(v)
	case *types.Pointer:
		switch v := v.Elem().Underlying().(type) {
		case *types.Struct:
			s = FromType(v)
		}
	}
	if s == nil {
		log.Printf("warning: struct.New() on non-struct value (type:%s)",
			v.Type().String())
		return nil
	}
	s.ns = scope
	if v, ok := v.(ssa.Value); ok {
		s.Value = v
	}
	return s
}

// a Field is a store.Key which also contain a pointer to its parent struct.
type Field store.Key

// SField is a wrapper for struct Field.
type SField struct {
	store.Key
	Struct *Struct
	Index  int
}

// Name returns the Name of the field if defined, otherwise return a synthetic
// field in the form of "struct_fieldindex"
func (f SField) Name() string {
	if f.Key == nil {
		return fmt.Sprintf("%s_%d", f.Struct.Name(), f.Index)
	}
	return f.Key.Name()
}

func (f SField) Type() types.Type {
	if f.Key == nil {
		switch t := f.Struct.Type().Underlying().(type) {
		case *types.Pointer:
			return t.Elem().Underlying().(*types.Struct).Field(f.Index).Type()
		case *types.Struct:
			return t.Field(f.Index).Type()
		default:
			panic(fmt.Sprintf("Parent struct of SField is not *types.Struct (type:%s)",
				f.Struct.Type().Underlying().String()))
		}
	}
	return f.Key.Type()
}

func (f SField) String() string {
	if f.Key == nil {
		return fmt.Sprintf("<nil>:%s[#%d]", f.Struct.Name(), f.Index)
	}
	return f.Key.String()
}

func (f SField) Pos() token.Pos {
	if f.Key == nil {
		return token.NoPos
	}
	return f.Key.Pos()
}

// empty is a placeholder ssa.Value for unused structs.
// It can be safely overwritten by concrete allocation/usage of the struct.
type emptyStruct struct {
	t *types.Struct
}

func (v emptyStruct) Name() string                  { return "_empty_struct_" }
func (v emptyStruct) Parent() *ssa.Function         { return nil }
func (v emptyStruct) Pos() token.Pos                { return token.NoPos }
func (v emptyStruct) Type() types.Type              { return v.t }
func (v emptyStruct) Referrers() *[]ssa.Instruction { return nil }
func (v emptyStruct) String() string                { return fmt.Sprintf("_empty_struct%d_", v.t.NumFields()) }

// emptyField is a placeholder field for unused struct fields (store.Key).
// It is used for temporary holding struct field when passed as a parameter.
type emptyField struct {
	T     types.Type
	Index int
}

func (f emptyField) Name() string { return "_" }

func (f emptyField) Pos() token.Pos {
	panic("not implemented")
}

func (f emptyField) String() string {
	panic("not implemented")
}

func (f emptyField) Type() types.Type {
	panic("not implemented")
}

// FromType creates a Struct from only the type information, useful for making
// placeholder structs where the number of fields in the struct is needed.
func FromType(t *types.Struct) *Struct {
	s := Struct{
		Value:  emptyStruct{t: t},
		ns:     nil,
		Fields: make([]Field, t.NumFields()),
	}
	return &s
}

func (s *Struct) UniqName() string {
	return fmt.Sprintf("%s.%s_struct%d", s.ns.UniqName(), s.Value.Name(), len(s.Fields))
}

// Expand of a Struct traverses (recursively) all its fields and return a slice
// containing the store.Key to each of their fields.
//
// The input Struct is expected to be filled with field keys in the current
// scope, any field left empty (nil:store.Key) are treated as undefined (zero
// value). The exception is when the type of the empty value is also a
// "go/types".Struct type, in which case empty fields are added as part of the
// return value. It is designed such that Expand of the same type will result in
// slices with the same length.
//
// Example:
//
//  struct {         // t0 = Alloc(...)
//  	X int        // t1 = Field(t0, 1)
//  	Y struct {   // t2 = Field(t0, 2)
//  		Z byte   // ; unused
//  		A string // t3 = Field(t2, 2)
//  	}
//  }
//
// if x is ssa.Value t0, Struct(t0).Expand() would become
//
//   []store.Key{t0, t1 /*t0_0*/, t2 /*t0_1*/, SField(nil) /*t2_0*/, t3 /*t2_2*/}
//
// Fields are always wrapped with SField.
func (s *Struct) Expand() []store.Key {
	fields := []store.Key{s}
	for i, field := range s.Fields {
		if sfield, ok := field.(SField); ok {
			// If field is already wrapped in SField, use it.
			fields = append(fields, sfield)
		} else {
			// Use SField to keep the struct hierarchy information.
			fields = append(fields, SField{Key: field, Struct: s, Index: i})
		}

		// Use types to expand.
		switch structType := s.Value.Type().Underlying().(type) {
		case *types.Struct:
			fieldType := structType.Field(i).Type().Underlying()
			if structFieldType, ok := fieldType.(*types.Struct); ok {
				var s *Struct
				if field != nil {
					if sfield, ok := field.(SField); ok {
						if sfield.Key != nil {
							// Need to unwrap field to get the struct.
							s = sfield.Key.(*Struct)
						}
					} else {
						// Field is a struct with predefined Field entry.
						s = field.(*Struct)
					}
				}
				if s != nil {
					fields = append(fields, s.Expand()...)
				} else {
					// Field is a struct but not defined.
					fields = append(fields, FromType(structFieldType).Expand()...)
				}
			}
		case *types.Pointer:
			switch structType := structType.Elem().Underlying().(type) {
			case *types.Struct:
				fieldType := structType.Field(i).Type().Underlying()
				if structFieldType, ok := fieldType.(*types.Struct); ok {
					if field != nil {
						// Field is a struct with predefined Field entry.
						// log.Printf("*Field is defined as %#v", field)
					}
					// Field is a struct but not defined.
					fields = append(fields, FromType(structFieldType).Expand()...)
				}
			}
		}
	}
	return fields
}
