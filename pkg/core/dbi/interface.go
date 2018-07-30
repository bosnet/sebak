package dbi

type DatabaseWriter interface {
	Put(key []byte, value []byte) error
}

type DatabaseReader interface {
	Get(key []byte) ([]byte, error)

	Has(key []byte) (bool, error)
}
