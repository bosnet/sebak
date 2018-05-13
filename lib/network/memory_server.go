package network

import (
	"context"

	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/util"
)

type MemoryTransport struct {
	ctx        context.Context
	endpoint   *util.Endpoint
	connWriter chan Message

	receiveChannel chan Message
}

func (t *MemoryTransport) Context() context.Context {
	return t.ctx
}

func (t *MemoryTransport) SetContext(ctx context.Context) {
	t.ctx = ctx
}

func (p *MemoryTransport) Endpoint() *util.Endpoint {
	return p.endpoint
}

func (p *MemoryTransport) Start() error {
	defer close(p.connWriter)

	p.receiveMessage()

	return nil
}

func (p *MemoryTransport) Ready() error {
	return nil
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
		case d := <-p.connWriter:
			p.receiveChannel <- d
		}
	}
}

func NewMemoryTransport() *MemoryTransport {
	n := &MemoryTransport{
		endpoint:       &util.Endpoint{Scheme: "memory", Host: uuid.New().String()},
		connWriter:     make(chan Message),
		receiveChannel: make(chan Message),
	}

	return n
}
