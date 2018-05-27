package storage

import "os"

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
