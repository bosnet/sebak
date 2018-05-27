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
	if err = st.Init(map[string]string{"path": "_memory_"}); err != nil {
		return
	}

	return
}
