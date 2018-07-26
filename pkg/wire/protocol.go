package wire

import (
	"io"
)

type MsgId [4]byte

type Protocol interface {
	Register(MsgId, interface{})

	Pack(io.Writer, interface{}) (int, error)

	Unpack(io.Reader) (interface{}, error)
}
