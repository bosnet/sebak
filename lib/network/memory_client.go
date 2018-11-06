package network

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
)

type MemoryTransportClient struct {
	endpoint *common.Endpoint

	server *MemoryNetwork
}

func NewMemoryNetworkClient(endpoint *common.Endpoint, server *MemoryNetwork) *MemoryTransportClient {
	return &MemoryTransportClient{
		endpoint: endpoint,
		server:   server,
	}
}

func (m *MemoryTransportClient) Endpoint() *common.Endpoint {
	return m.endpoint
}

func (m *MemoryTransportClient) Connect(node node.Node) (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) GetNodeInfo() (b []byte, err error) {
	b = m.server.GetNodeInfo()
	return
}

func (m *MemoryTransportClient) SendMessage(message common.Serializable) (body []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(common.TransactionMessage, s)

	return
}

func (m *MemoryTransportClient) SendTransaction(message common.Serializable) (body []byte, err error) {
	return m.SendMessage(message)
}

func (m *MemoryTransportClient) SendBallot(message common.Serializable) (body []byte, err error) {
	var s []byte
	if s, err = message.Serialize(); err != nil {
		return
	}
	m.server.Send(common.BallotMessage, s)

	return
}

func (m *MemoryTransportClient) GetTransactions([]string) ([]byte, error) {
	return []byte{}, errors.NotImplemented
}
