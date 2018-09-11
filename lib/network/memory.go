package network

import (
	"net"
	"net/http"

	"boscoin.io/sebak/lib/common"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var memoryNetworks map[ /* endpoint */ string]*MemoryNetwork

func init() {
	memoryNetworks = map[string]*MemoryNetwork{}
}

func CleanUpMemoryNetwork() {
	// BUG(osx): `CleanUpMemoryNetwork` causes 'runtime error: invalid memory address or nil pointer dereference'
	//MemoryNetworks = map[string]*MemoryNetworks{}
}

func addMemoryNetwork(m *MemoryNetwork) {
	memoryNetworks[m.Endpoint().String()] = m
}

type MemoryNetwork struct {
	localNode  common.Serializable
	endpoint   *common.Endpoint
	connWriter chan common.NetworkMessage
	close      chan bool

	receiveChannel chan common.NetworkMessage
}

func (t *MemoryNetwork) GetClient(endpoint *common.Endpoint) NetworkClient {
	n, ok := memoryNetworks[endpoint.String()]
	if !ok {
		return nil
	}

	return NewMemoryNetworkClient(endpoint, n)
}

func (p *MemoryNetwork) AddWatcher(f func(Network, net.Conn, http.ConnState)) {
	return
}
func (p *MemoryNetwork) Endpoint() *common.Endpoint {
	return p.endpoint
}

func (p *MemoryNetwork) Start() error {
	defer close(p.connWriter)

	p.receiveMessage()

	return nil
}

func (p *MemoryNetwork) Stop() {
	p.close <- true
}

func (p *MemoryNetwork) Ready() error {
	return nil
}

func (p *MemoryNetwork) SetMessageBroker(MessageBroker) {
}

func (p *MemoryNetwork) MessageBroker() MessageBroker {
	return nil
}

func (p *MemoryNetwork) IsReady() bool {
	return true
}

func (p *MemoryNetwork) GetNodeInfo() []byte {
	o, _ := p.localNode.Serialize()
	return o
}

func (p *MemoryNetwork) Send(mt common.MessageType, b []byte) (err error) {
	p.connWriter <- common.NewNetworkMessage(mt, b)

	return
}

func (p *MemoryNetwork) ReceiveChannel() chan common.NetworkMessage {
	return p.receiveChannel
}

func (p *MemoryNetwork) ReceiveMessage() <-chan common.NetworkMessage {
	return p.receiveChannel
}

func (p *MemoryNetwork) receiveMessage() {
	for {
		select {
		case <-p.close:
			break
		case d := <-p.connWriter:
			p.receiveChannel <- d
		}
	}
}

func (p *MemoryNetwork) SetLocalNode(localNode common.Serializable) {
	p.localNode = localNode
}

func CreateNewMemoryEndpoint() *common.Endpoint {
	return &common.Endpoint{Scheme: "memory", Host: uuid.New().String()}
}

func NewMemoryNetwork() *MemoryNetwork {
	n := &MemoryNetwork{
		endpoint:       CreateNewMemoryEndpoint(),
		connWriter:     make(chan common.NetworkMessage),
		receiveChannel: make(chan common.NetworkMessage),
		close:          make(chan bool),
	}

	addMemoryNetwork(n)

	return n
}

func (p *MemoryNetwork) AddHandler(string, http.HandlerFunc) *mux.Route {
	return &mux.Route{}
}
