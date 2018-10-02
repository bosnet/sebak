package sync

import (
	"context"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	networkID := []byte("test-network")
	_, nw, _ := network.CreateMemoryNetwork(nil)

	v := NewBlockValidator(nw, st, networkID)

	ctx := context.Background()

	bk := block.TestMakeNewBlock(nil)
	si := &SyncInfo{
		BlockHeight: uint64(1),
		Block:       &bk,
	}

	err := v.Validate(ctx, si)
	require.Nil(t, err)

}
