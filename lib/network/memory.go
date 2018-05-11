package network

import (
	"net/url"

	"github.com/google/uuid"
)

type MemNetwork struct {
	endpoint   *url.URL
	connWriter chan TransportMessage

	receiveChannel chan TransportMessage
}

func (p *MemNetwork) Endpoint() *url.URL {
	return p.endpoint
}

func (p *MemNetwork) Start() error {
	defer close(p.connWriter)

	p.receiveMessage()

	return nil
}

func (p *MemNetwork) Ready() error {
	return nil
}

func (p *MemNetwork) Send(mt TransportMessageType, b []byte) (err error) {
	p.connWriter <- NewTransportMessage(mt, b)

	return
}

func (p *MemNetwork) ReceiveChannel() chan TransportMessage {
	return p.receiveChannel
}

func (p *MemNetwork) ReceiveMessage() <-chan TransportMessage {
	return p.receiveChannel
}

func (p *MemNetwork) receiveMessage() {
	for {
		select {
		case d := <-p.connWriter:
			p.receiveChannel <- d
		}
	}
}

func NewMemNetwork() *MemNetwork {
	n := &MemNetwork{
		endpoint:       &url.URL{Scheme: "memory", Host: uuid.New().String()},
		connWriter:     make(chan TransportMessage),
		receiveChannel: make(chan TransportMessage),
	}

	return n
}
