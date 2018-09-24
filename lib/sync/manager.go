package sync

import (
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"
)

var _ Producer = (*Manager)(nil)
var _ Stopper = (*Manager)(nil)

type Manager struct {
	fetcherLayer    Fetcher
	validationLayer Validator

	retryTimeout  time.Duration
	checkInterval time.Duration

	storage *storage.LevelDBBackend

	messages chan *Message
	response <-chan *Response

	stop chan chan struct{}
}

func (m *Manager) Stop() error {
	c := make(chan struct{})
	m.stop <- c
	<-c
	return nil
}

func (m *Manager) SetResponse(resp <-chan *Response) error {
	m.response = resp
	return nil
}

func (m *Manager) Produce() <-chan *Message {
	return m.messages
}

func (m *Manager) loop() {
	timer := time.NewTimer(m.checkInterval)
	var syncBlockHeight uint64
	for {
		select {
		case <-timer.C:
			blk, err := block.GetLatestBlock(m.storage)
			if err != nil {
				//TODO: logging
				continue
			}
			newHeight := blk.Height + 1
			if newHeight > syncBlockHeight {
				msg := &Message{
					BlockHeight: newHeight,
				}
				m.messages <- msg
				syncBlockHeight = newHeight
			}
		case resp := <-m.response:
			time.AfterFunc(m.retryTimeout, func() {
				m.messages <- resp.Message()
			})
		case c := <-m.stop:
			close(c)
			return
		}
	}
}
