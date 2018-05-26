package sebaknetwork

import (
	"context"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/common"
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

func getMemoryNetwork(endpoint *sebakcommon.Endpoint) *MemoryNetwork {
	n, _ := memoryNetworks[endpoint.String()]
	return n
}

type MemoryNetwork struct {
	ctx        context.Context
	endpoint   *sebakcommon.Endpoint
	connWriter chan Message
	close      chan bool

	receiveChannel chan Message
}

func (t *MemoryNetwork) Context() context.Context {
	return t.ctx
}

func (t *MemoryNetwork) SetContext(ctx context.Context) {
	t.ctx = ctx
}

func (t *MemoryNetwork) GetClient(endpoint *sebakcommon.Endpoint) NetworkClient {
	n, ok := memoryNetworks[endpoint.String()]
	if !ok {
		return nil
	}

	return NewMemoryNetworkClient(endpoint, n)
}

func (p *MemoryNetwork) AddWatcher(f func(Network, net.Conn, http.ConnState)) {
	return
}
func (p *MemoryNetwork) Endpoint() *sebakcommon.Endpoint {
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

func (p *MemoryNetwork) IsReady() bool {
	return true
}

func (p *MemoryNetwork) Send(mt MessageType, b []byte) (err error) {
	p.connWriter <- NewMessage(mt, b)

	return
}

func (p *MemoryNetwork) ReceiveChannel() chan Message {
	return p.receiveChannel
}

func (p *MemoryNetwork) ReceiveMessage() <-chan Message {
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

func (p *MemoryNetwork) GetNodeInfo() []byte {
	currentNode := p.Context().Value("currentNode").(sebakcommon.Serializable)
	o, _ := currentNode.Serialize()
	return o
}

func CreateNewMemoryEndpoint() *sebakcommon.Endpoint {
	return &sebakcommon.Endpoint{Scheme: "memory", Host: uuid.New().String()}
}

func NewMemoryNetwork() *MemoryNetwork {
	n := &MemoryNetwork{
		endpoint:       CreateNewMemoryEndpoint(),
		connWriter:     make(chan Message),
		receiveChannel: make(chan Message),
		close:          make(chan bool),
	}

	addMemoryNetwork(n)

	return n
}
