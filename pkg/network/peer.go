package network

import (
	"boscoin.io/sebak/pkg/wire/message"
	"io"
)

type Peer interface {
	io.Writer

	Id() message.PeerId

	IsInbound() bool

	RemoteAddr() string

	Close() error
}
