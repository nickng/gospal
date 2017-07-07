package migoinfer

// Utility helper functions.

import (
	"go/types"

	"github.com/nickng/gospal/store"
)

func isChan(k store.Key) bool {
	switch t := k.Type().Underlying().(type) {
	case *types.Chan:
		return true
	case *types.Pointer:
		switch t.Elem().Underlying().(type) {
		case *types.Chan:
			return true
		}
	}
	return false
}

func isStruct(k store.Key) bool {
	switch t := k.Type().Underlying().(type) {
	case *types.Struct:
		return true
	case *types.Pointer:
		// only pointer to struct is 'Struct'
		switch t.Elem().Underlying().(type) {
		case *types.Struct:
			return true
		}
	}
	return false
}
