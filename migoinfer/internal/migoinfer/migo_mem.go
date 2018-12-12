package migoinfer

import (
	"github.com/nickng/gospal/store"
	"github.com/nickng/migo/v3"
)

func migoNewMem(mem store.Value) migo.Statement {
	return &migo.NewMem{Name: mem.UniqName()}
}

func migoRead(mem store.Value) migo.Statement {
	return &migo.MemRead{Name: mem.UniqName()}
}

func migoWrite(mem store.Value) migo.Statement {
	return &migo.MemWrite{Name: mem.UniqName()}
}
