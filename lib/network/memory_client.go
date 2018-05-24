package network

import (
	"github.com/spikeekips/sebak/lib/util"
)

type MemoryTransportClient struct {
	endpoint *util.Endpoint

	server *MemoryTransport
}

func NewMemoryTransportClient(endpoint *util.Endpoint, server *MemoryTransport) *MemoryTransportClient {
	return &MemoryTransportClient{
		endpoint: endpoint,
		server:   server,
	}
}

func (m *MemoryTransportClient) Endpoint() *util.Endpoint {
	return m.endpoint
}

func (m *MemoryTransportClient) Connect(node util.Node) (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) GetNodeInfo() (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) SendMessage(message util.Serializable) (err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(MessageFromClient, s)

	return
}

func (m *MemoryTransportClient) SendBallot(message util.Serializable) (err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(BallotMessage, s)

	return
}
