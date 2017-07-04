package store

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

func abbrv(s string) string {
	fields := strings.Fields(s)
	if len(fields) > 0 {
		return strings.ToLower(fields[0])
	}
	return ""
}

// Special types and values

// A MockKey is a placeholder for a Key.
// MockKey requires a type to be specified.
// A MockKey is a wrapper, and should not be modified after creation, so all
// methods takes a value receiver.
type MockKey struct {
	Typ         types.Type
	SrcPos      token.Pos
	Description string // Description of the key, start with a verb to get a descriptive Name()
}

func (k MockKey) Name() string     { return fmt.Sprintf("_%s_", abbrv(k.Description)) }
func (k MockKey) Pos() token.Pos   { return k.SrcPos }
func (k MockKey) Type() types.Type { return k.Typ }
func (k MockKey) String() string   { return fmt.Sprintf("[_%s_:%s]", k.Description, k.Type().String()) }

// A MockValue is a placeholder for a Value.
//
// This could mean the value comes from external (C), or is not instantiated in
// main body of program.
type MockValue struct {
	SrcPos      token.Pos
	Description string // Description of the key
}

func (v MockValue) UniqName() string {
	return fmt.Sprintf("%s_%v", strings.Join(strings.Fields(v.Description), "_"), v.SrcPos)
}

// Unused is an annotated MockKey indicating the Key (local variable) is unused.
type Unused struct {
	MockKey
}
