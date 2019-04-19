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

func migoNewMutex(mu store.Value) migo.Statement {
	return &migo.NewSyncMutex{Name: mu.UniqName()}
}

func migoLock(mu store.Value) migo.Statement {
	return &migo.SyncMutexLock{Name: mu.UniqName()}
}

func migoUnlock(mu store.Value) migo.Statement {
	return &migo.SyncMutexUnlock{Name: mu.UniqName()}
}

func migoNewRWMutex(mu store.Value) migo.Statement {
	return &migo.NewSyncRWMutex{Name: mu.UniqName()}
}

func migoRLock(mu store.Value) migo.Statement {
	return &migo.SyncRWMutexRLock{Name: mu.UniqName()}
}

func migoRUnlock(mu store.Value) migo.Statement {
	return &migo.SyncRWMutexRUnlock{Name: mu.UniqName()}
}
