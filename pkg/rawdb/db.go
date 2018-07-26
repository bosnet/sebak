package rawdb

type Database interface {
	Get(key []byte) ([]byte, error)

	Has(key []byte) (bool, error)

	Put(key []byte, value []byte) error

	Delete(key []byte) error
}
