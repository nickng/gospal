package migoinfer

import (
	"fmt"
	"go/token"

	"github.com/pkg/errors"
)

var (
	ErrFnIsNil      = errors.New("function is nil")
	ErrIncompatType = errors.New("incompatible type")
)

type ErrChanBufSzNonStatic struct {
	Pos token.Position
}

func (e ErrChanBufSzNonStatic) Error() string {
	return fmt.Sprintf("%s: channel buffer size is not constant", e.Pos.String())
}
