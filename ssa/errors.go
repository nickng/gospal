package ssa

import "errors"

var (
	ErrNoTestMainPkgs = errors.New("no main packages in tests")
	ErrNoMainPkgs     = errors.New("no main packages")
)
