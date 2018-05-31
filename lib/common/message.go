package sebakcommon

type Message interface {
	GetType() string
	GetHash() string
	Serialize() ([]byte, error)
	String() string
	IsWellFormed([]byte) error
	Equal(Message) bool
	// Validate(sebakstorage.LevelDBBackend) error
}
