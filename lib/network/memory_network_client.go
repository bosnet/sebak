package network

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type MemoryTransportClient struct {
	endpoint *sebakcommon.Endpoint

	server *MemoryNetwork
}

func NewMemoryNetworkClient(endpoint *sebakcommon.Endpoint, server *MemoryNetwork) *MemoryTransportClient {
	return &MemoryTransportClient{
		endpoint: endpoint,
		server:   server,
	}
}

func (m *MemoryTransportClient) Endpoint() *sebakcommon.Endpoint {
	return m.endpoint
}

func (m *MemoryTransportClient) Connect(node sebaknode.Node) (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) GetNodeInfo() (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) SendMessage(message sebakcommon.Serializable) (body []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(TransactionMessage, s)

	return
}

func (m *MemoryTransportClient) SendBallot(message sebakcommon.Serializable) (body []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(BallotMessage, s)

	return
}
