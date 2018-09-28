package sync

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()
	_, nw, _ := network.CreateMemoryNetwork(nil)
	msgc := make(chan *Message)

	v := NewBlockValidator(nw, st)
	v.Consume(msgc)
	defer v.Stop()

	respc := v.Response()

	bk := block.TestMakeNewBlock([]string{})
	bk.Height = uint64(1)

	msg := &Message{
		BlockHeight: bk.Height,
		Block:       &bk,
	}
	msgc <- msg
	resp := <-respc
	require.Nil(t, resp.Err())

	ok, err := block.ExistsBlockByHeight(st, bk.Height)
	require.Nil(t, err)
	require.Equal(t, ok, true)
}
