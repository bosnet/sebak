package sync

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	tickc := make(chan time.Time)
	c := NewMockConsumer()
	p := NewMockProcessor()
	st := storage.NewTestStorage()

	//TODO(anarhcer): NewTestManager and NewManager?
	m := &Manager{
		fetcherLayer:    p,
		validationLayer: c,

		storage: st,

		afterFunc: func(d time.Duration) <-chan time.Time {
			return tickc
		},

		messages: make(chan *Message),
		response: make(chan *Response),

		stopLoop: make(chan chan struct{}),
		stopResp: make(chan chan struct{}),
	}
	defer m.Stop()

	Pipeline(m, p)
	Pipeline(p, c)

	go func() {
		m.Run()
	}()

	{
		msg := <-c.Message()
		require.Equal(t, msg.BlockHeight, uint64(1))
	}

	bk := block.TestMakeNewBlock([]string{})
	bk.Height = uint64(1)
	require.Nil(t, bk.Save(st))

	tickc <- time.Time{}
	{
		msg := <-c.Message()
		require.Equal(t, msg.BlockHeight, uint64(2))
	}
}
