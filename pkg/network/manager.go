package network

import (
	"boscoin.io/sebak/pkg/wire/message"
	"net/url"
)

type Manager interface {
	Start()

	Stop()

	Connect(*url.URL) error

	Send(message.PeerId, interface{}) error

	Broadcast(interface{}) error

	Peers() Peerstore
}
