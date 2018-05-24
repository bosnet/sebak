package network

import (
	"context"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/util"
)

var memServers map[ /* endpoint */ string]*MemoryTransport

func init() {
	memServers = map[string]*MemoryTransport{}
}

func cleanUpMemoryServer() {
	memServers = map[string]*MemoryTransport{}
}

func addMemoryServer(server *MemoryTransport) {
	memServers[server.Endpoint().String()] = server
}

func getMemoryServer(endpoint *util.Endpoint) *MemoryTransport {
	server, _ := memServers[endpoint.String()]
	return server
}

type MemoryTransport struct {
	ctx        context.Context
	endpoint   *util.Endpoint
	connWriter chan Message
	close      chan bool

	receiveChannel chan Message
}

func (t *MemoryTransport) Context() context.Context {
	return t.ctx
}

func (t *MemoryTransport) SetContext(ctx context.Context) {
	t.ctx = ctx
}

func (t *MemoryTransport) GetClient(endpoint *util.Endpoint) TransportClient {
	server, ok := memServers[endpoint.String()]
	if !ok {
		return nil
	}

	return NewMemoryTransportClient(endpoint, server)
}

func (p *MemoryTransport) AddWatcher(f func(TransportServer, net.Conn, http.ConnState)) {
	return
}
func (p *MemoryTransport) Endpoint() *util.Endpoint {
	return p.endpoint
}

func (p *MemoryTransport) Start() error {
	defer close(p.connWriter)

	p.receiveMessage()

	return nil
}

func (p *MemoryTransport) Stop() {
	p.close <- true
}

func (p *MemoryTransport) Ready() error {
	return nil
}

func (p *MemoryTransport) IsReady() bool {
	return true
}

func (p *MemoryTransport) Send(mt MessageType, b []byte) (err error) {
	p.connWriter <- NewMessage(mt, b)

	return
}

func (p *MemoryTransport) ReceiveChannel() chan Message {
	return p.receiveChannel
}

func (p *MemoryTransport) ReceiveMessage() <-chan Message {
	return p.receiveChannel
}

func (p *MemoryTransport) receiveMessage() {
	for {
		select {
		case <-p.close:
			break
		case d := <-p.connWriter:
			p.receiveChannel <- d
		}
	}
}

func (p *MemoryTransport) GetNodeInfo() []byte {
	currentNode := p.Context().Value("currentNode").(util.Serializable)
	o, _ := currentNode.Serialize()
	return o
}

func CreateNewMemoryEndpoint() *util.Endpoint {
	return &util.Endpoint{Scheme: "memory", Host: uuid.New().String()}
}

func NewMemoryTransport() *MemoryTransport {
	n := &MemoryTransport{
		endpoint:       CreateNewMemoryEndpoint(),
		connWriter:     make(chan Message),
		receiveChannel: make(chan Message),
		close:          make(chan bool),
	}

	addMemoryServer(n)

	return n
}
