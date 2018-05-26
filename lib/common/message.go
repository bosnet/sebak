package sebakcommon

type Message interface {
	GetType() string
	GetHash() string
	Serialize() ([]byte, error)
	String() string
	IsWellFormed() error
	Equal(Message) bool
	// Validate(storage.LevelDBBackend) error
}
