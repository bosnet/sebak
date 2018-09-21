package sync

import "boscoin.io/sebak/lib/block"

type Producer interface {
	Produce(<-chan *Response)
	Message() <-chan *Message
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

type Stopper interface {
	Stop() error
}

type FetchLayer interface {
	Processor
	Stopper
}

type ValidationLayer interface {
	Consumer
	Stopper
}
