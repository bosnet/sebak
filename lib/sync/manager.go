package sync

import (
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
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
	cancel   chan chan struct{}

	logger log15.Logger
}

func (m *Manager) Run() error {
	m.loop()
	return nil
}

func (m *Manager) Stop() error {
	m.logger.Debug("Stopping")
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
	m.logger.Info("Stopped")
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
	cancel := make(chan struct{}, 1)

	for {
		select {
		case <-checkc:
			syncBlockHeight = m.checkBlockHeight(syncBlockHeight)
			checkc = m.afterFunc(m.checkInterval)
		case resp := <-m.response:
			if resp.Err() == nil {
				continue
			}
			go func() {
				msg := resp.Message()
				m.logger.Info("Retry response", "height", msg.BlockHeight)

				currBlockHeight := m.currBlockHeight()
				if currBlockHeight > 0 && msg.BlockHeight <= currBlockHeight {
					m.logger.Info("sync block height is old", "sync/height", msg.BlockHeight, "db/height", currBlockHeight)
					return
				}

				retryc := m.afterFunc(m.retryInterval)
				select {
				case <-retryc:
				case <-cancel:
					return
				}
				select {
				case m.messages <- resp.Message():
				case <-cancel:
					return
				}
			}()
		case c := <-m.cancel:
			close(c)
			return
		case c := <-m.stopLoop:
			close(cancel)
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

		select {
		case m.messages <- msg:
		case c := <-m.stopLoop:
			m.cancel <- c
		}
		m.logger.Info("Starting sync", "height", newHeight)
		return newHeight
	}
	return height
}

func (m *Manager) currBlockHeight() uint64 {
	blk, err := block.GetLatestBlock(m.storage)
	if err != nil {
		m.logger.Error("block.GetLatestBlock", "err", err)
		return 0
	}
	return blk.Height
}
