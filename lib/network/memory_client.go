package network

import (
	"github.com/spikeekips/sebak/lib/util"
)

type MemotyTransportClient struct {
	endpoint *util.Endpoint

	server *MemoryTransport
}

func (m *MemotyTransportClient) Endpoint() *util.Endpoint {
	return m.endpoint
}

func (m *MemotyTransportClient) GetNodeInfo() (b []byte, err error) {
	return
}

func (m *MemotyTransportClient) SendMessage(message util.Serializable) (b []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(MessageFromClient, s)

	return
}

func (m *MemotyTransportClient) SendBallot(message util.Serializable) (b []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(BallotMessage, s)

	return
}
