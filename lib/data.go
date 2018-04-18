package sebak

type Serializable interface {
	Serialize() ([]byte, error)
}
