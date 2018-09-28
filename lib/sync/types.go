package sync

import (
	"net/http"

	"boscoin.io/sebak/lib/block"
)

type Stopper interface {
	Stop() error
}

type Producer interface {
	Produce() <-chan *Message
	SetResponse(<-chan *Response) error
}

type Consumer interface {
	Consume(<-chan *Message) error
	Response() <-chan *Response
}

type Processor interface {
	Producer
	Consumer
}

type Message struct {
	BlockHeight uint64
	Block       *block.Block
	Txs         []*block.BlockTransaction
	Ops         []*block.BlockOperation
}

type Response struct {
	err error
	msg *Message
}

func (r Response) Err() error {
	return r.err
}
func (r Response) Message() *Message {
	return r.msg
}

type Fetcher interface {
	Processor
	Stopper
}

type Validator interface {
	Consumer
	Stopper
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}
