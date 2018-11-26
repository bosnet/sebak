package network

import (
	"io"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type MemoryNetwork struct {
	localNode  *node.LocalNode
	endpoint   *common.Endpoint
	connWriter chan common.NetworkMessage
	close      chan bool

	receiveChannel chan common.NetworkMessage
	// They all share the same map to find each other
	peers map[ /* endpoint */ string]*MemoryNetwork

	messageBroker MessageBroker
}

func (t *MemoryNetwork) GetClient(endpoint *common.Endpoint) NetworkClient {
	n, ok := t.peers[endpoint.String()]
	if !ok {
		panic("Trying to get inexistant client, this is a bug in the tests!")
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

func (p *MemoryNetwork) SetMessageBroker(mb MessageBroker) {
	p.messageBroker = mb
}

func (p *MemoryNetwork) MessageBroker() MessageBroker {
	return p.messageBroker
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

func (p *MemoryNetwork) SetLocalNode(localNode *node.LocalNode) {
	p.localNode = localNode
}

func CreateNewMemoryEndpoint() *common.Endpoint {
	return &common.Endpoint{Scheme: "memory", Host: uuid.New().String()}
}

func (prev *MemoryNetwork) NewMemoryNetwork() *MemoryNetwork {
	var peers map[string]*MemoryNetwork
	if prev != nil {
		peers = prev.peers
	} else {
		peers = make(map[string]*MemoryNetwork)
	}

	n := &MemoryNetwork{
		endpoint:       CreateNewMemoryEndpoint(),
		connWriter:     make(chan common.NetworkMessage),
		receiveChannel: make(chan common.NetworkMessage),
		close:          make(chan bool),
		peers:          peers,
	}

	n.peers[n.endpoint.String()] = n
	n.messageBroker = &MemoryMessageBroker{network: n}

	return n
}

func (p *MemoryNetwork) AddHandler(string, http.HandlerFunc) *mux.Route {
	return &mux.Route{}
}

func (p *MemoryNetwork) AddMiddleware(string, ...mux.MiddlewareFunc) error {
	return nil
}

type MemoryMessageBroker struct {
	network  *MemoryNetwork
	Messages []common.NetworkMessage
}

func (r *MemoryMessageBroker) Response(w io.Writer, o []byte) error {
	_, err := w.Write(o)
	return err
}

func (r *MemoryMessageBroker) Receive(m common.NetworkMessage) {
	r.network.ReceiveChannel() <- m
}
