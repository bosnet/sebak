package storage

import (
	"fmt"
	"os"
)

func CleanDB(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	os.RemoveAll(path)

	return
}

func NewTestMemoryLevelDBBackend() (st *LevelDBBackend, err error) {
	st = &LevelDBBackend{}
	config, _ := NewConfigFromString("memory://")
	if err = st.Init(config); err != nil {
		return
	}

	return
}

func NewTestFileLevelDBBackend(f string) (st *LevelDBBackend, err error) {
	st = &LevelDBBackend{}
	config, _ := NewConfigFromString(fmt.Sprintf("file://%s", f))
	if err = st.Init(config); err != nil {
		return
	}

	return
}
