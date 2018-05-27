package storage

// TODO use URI
type Config map[string]string

type IterItem struct {
	N     int64
	Key   []byte
	Value []byte
}

type Item struct {
	Key   string
	Value interface{}
}

type Model struct {
}

func NewStorage(config Config) (st *LevelDBBackend, err error) {
	st = &LevelDBBackend{}
	if err = st.Init(config); err != nil {
		return
	}

	return
}
