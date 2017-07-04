package store

import (
	"go/types"
	"testing"
)

func TestMockKey(t *testing.T) {
	unused1 := MockKey{Typ: types.NewInterface(nil, nil), Description: "Unused value"}
	unused2 := MockKey{Typ: types.NewInterface(nil, nil), Description: "Unused value"}
	empty := MockKey{Typ: types.NewInterface(nil, nil)}
	if want, got := `_unused_`, unused1.Name(); want != got {
		t.Errorf("MockKey should abbrv. description as Name()\nwant: %s\ngot: %s\n", want, got)
	}
	if want, got := `__`, empty.Name(); want != got {
		t.Errorf("MockKey should abbrv. description as Name()\nwant: %s\ngot: %s\n", want, got)
	}
	if unused1 == unused2 {
		t.Errorf("cannot distinguish between MockKeys with same parameters\n%#v\n%#v\n",
			unused1, unused2)
	}
	var (
		_ Key = unused1 // Static check that MockKey implements Key.
		_ Key = unused2 // Static check that MockKey implements Key.
		_ Key = empty   // Static check that MockKey implements Key.
	)
}
