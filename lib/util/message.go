package util

type Message interface {
	GetType() string
	GetHash() string
	Serialize() ([]byte, error)
	String() string
	IsWellFormed() error
	// Validate(storage.LevelDBBackend) error
}
