package sync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	var wg sync.WaitGroup
	msg := make(chan *Message)

	c := NewMockConsumer()
	c.Consume(msg)
	defer c.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := <-c.Message()
		require.NotNil(t, msg)
		require.Equal(t, msg.BlockHeight, uint64(1))
	}()

	msg <- &Message{
		BlockHeight: 1,
	}

	wg.Wait()
}
