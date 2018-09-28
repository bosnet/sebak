package sync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipeline(t *testing.T) {
	p := NewMockProducer()
	c := NewMockConsumer()
	defer p.Stop()
	defer c.Stop()

	err := Pipeline(p, c)
	require.Nil(t, err)

	go func() {
		p.msgc <- &Message{}
	}()

	msg := <-c.Message()
	require.NotNil(t, msg)
}
