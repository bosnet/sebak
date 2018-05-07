package storage

type Config map[string]string

type Serializable interface {
	Serialize() ([]byte, error)
}

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
