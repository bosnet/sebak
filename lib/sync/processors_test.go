package sync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessor(t *testing.T) {
	var wg sync.WaitGroup
	msgc := make(chan *Message)

	p := NewMockProcessor()
	p.Consume(msgc)
	defer p.Stop()

	reqmsg := &Message{
		BlockHeight: 1,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		msgc := p.Produce()
		msg := <-msgc
		require.NotNil(t, msg)
		require.Equal(t, msg, reqmsg)
	}()

	msgc <- reqmsg

	wg.Wait()
}

func TestProcessors(t *testing.T) {
	var wg sync.WaitGroup
	msgc := make(chan *Message)

	p := NewMockProcessor()
	defer p.Stop()
	ps := NewProcessors(p)
	defer ps.Stop()

	ps.Consume(msgc)

	reqmsg := &Message{
		BlockHeight: 1,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := <-ps.Produce()
		require.NotNil(t, msg)
		require.Equal(t, msg, reqmsg)
	}()

	msgc <- reqmsg
	wg.Wait()
}
