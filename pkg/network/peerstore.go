package network

import (
	"boscoin.io/sebak/pkg/wire/message"
)

type Peerstore interface {
	Get(message.PeerId) (Peer, error)

	Put(message.PeerId, Peer) error

	List() []Peer
}
