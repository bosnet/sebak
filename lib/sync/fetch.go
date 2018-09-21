package sync

import (
	"context"
	"sync"
)

var _ FetchLayer = (*BlockFetch)(nil)

type BlockFetch struct {
	numWorker      int
	reqMessage     chan *Message
	consumeMessage <-chan *Message
	produceMessage chan *Message
	response       chan *Response

	stop       chan chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func NewBlockFetch(numWorker int) *BlockFetch {
	ctx, cancelFunc := context.WithCancel(context.Background())

	l := &BlockFetch{
		numWorker:      numWorker,
		reqMessage:     nil,
		produceMessage: make(chan *Message),
		stop:           make(chan chan struct{}),
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}
	go l.listenConsume()
	go l.runWorkers()

	return l
}

func (l *BlockFetch) Produce(response <-chan *Response) {
	res := <-response

	if res.Err() != nil {
		l.reqMessage <- res.Message()
	}
}

func (l *BlockFetch) Message() <-chan *Message {
	return l.produceMessage
}

func (l *BlockFetch) Response() <-chan *Response {
	return l.response
}

func (l *BlockFetch) Consume(msg <-chan *Message) error {
	l.consumeMessage = msg
	return nil
}

func (l *BlockFetch) Stop() error {
	c := make(chan struct{})
	l.stop <- c
	l.cancelFunc()
	l.wg.Wait()
	<-c
	return nil
}

func (l *BlockFetch) listenConsume() {
	for {
		select {
		case msg := <-l.consumeMessage:
			l.reqMessage <- msg
		case <-l.ctx.Done():
			l.wg.Done()
		}
	}
}

func (l *BlockFetch) runWorkers() {
	for i := 0; i < l.numWorker; i++ {
		go func() {
			l.wg.Add(1)
			for {
				select {
				case reqmsg := <-l.reqMessage:
					msg, err := l.fetch(reqmsg)
					resp := &Response{
						err: err,
					}
					l.produceMessage <- msg
					l.response <- resp
				case <-l.ctx.Done():
					l.wg.Done()
					return
				}
			}
		}()
	}

}

func (l *BlockFetch) fetch(msg *Message) (*Message, error) {
	//TODO(anarcher): Fetch block

	return msg, nil
}
