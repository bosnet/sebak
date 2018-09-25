package sync

import (
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"
)

var _ Producer = (*Manager)(nil)
var _ Stopper = (*Manager)(nil)

type AfterFunc = func(time.Duration) <-chan time.Time

type Manager struct {
	fetcherLayer    Fetcher
	validationLayer Validator

	retryInterval time.Duration
	checkInterval time.Duration

	afterFunc AfterFunc

	storage *storage.LevelDBBackend

	messages chan *Message
	response chan *Response

	stopLoop chan chan struct{}
	stopResp chan chan struct{}
}

func (m *Manager) Run() error {
	m.loop()
	return nil
}

func (m *Manager) Stop() error {
	{
		c := make(chan struct{})
		m.stopResp <- c
		<-c
	}
	{
		c := make(chan struct{})
		m.stopLoop <- c
		<-c
	}
	return nil
}

func (m *Manager) SetResponse(respc <-chan *Response) error {
	go func() {
		for {
			select {
			case resp := <-respc:
				m.response <- resp
			case s := <-m.stopResp:
				close(s)
				return
			}
		}
	}()
	return nil
}

func (m *Manager) Produce() <-chan *Message {
	return m.messages
}

func (m *Manager) loop() {
	checkc := m.afterFunc(m.checkInterval)
	syncBlockHeight := m.checkBlockHeight(0)
	for {
		select {
		case <-checkc:
			syncBlockHeight = m.checkBlockHeight(syncBlockHeight)
			checkc = m.afterFunc(m.checkInterval)
		case resp := <-m.response:
			go func() {
				retryc := m.afterFunc(m.retryInterval)
				<-retryc
				m.messages <- resp.Message()
			}()
		case c := <-m.stopLoop:
			close(c)
			return
		}
	}
}

func (m *Manager) checkBlockHeight(height uint64) uint64 {
	blk, err := block.GetLatestBlock(m.storage)
	if err != nil {
		//TODO: logging
	}
	newHeight := blk.Height + 1
	if newHeight > height {
		msg := &Message{
			BlockHeight: newHeight,
		}
		m.messages <- msg
		return newHeight
	}
	return height
}
