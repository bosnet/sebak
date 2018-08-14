package sebakstorage

type DBBackend interface {
	Has(string) (bool, error)
	Get(string, interface{}) error
	New(string, interface{}) error
	Set(string, interface{}) error
	Remove(string) error

	GetIterator(prefix string, reverse bool) (func() (IterItem, bool), func())

	News(...Item) error
	Sets(...Item) error

	Commit() error
	Discard() error
}
