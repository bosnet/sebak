package util

type Message interface {
	GetHash() string
	Serialize() ([]byte, error)
	String() string
	IsWellFormed() error
	// Validate(storage.LevelDBBackend) error
}
