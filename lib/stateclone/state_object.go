package stateclone

type DirtyStatus byte

const (
	StateObjectChanged DirtyStatus = iota
	StateObjectDeleted
)

type StateObject struct {
	Value interface{}
	State DirtyStatus
}
