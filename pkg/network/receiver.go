package network

import "boscoin.io/sebak/pkg/wire/message"

type Receiver interface {
	Start()

	Stop()

	OnConnect(id message.PeerId)

	Receive(id message.PeerId, msg interface{})
}
