package common

type Serializable interface {
	Serialize() ([]byte, error)
}
