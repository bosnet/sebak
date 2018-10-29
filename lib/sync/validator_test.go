package sync

import (
	"context"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	st := block.InitTestBlockchain()
	defer st.Close()

	networkID := []byte("test-network")
	_, nw, _ := network.CreateMemoryNetwork(nil)

	v := NewBlockValidator(nw, st, networkID, common.NewConfig())

	ctx := context.Background()

	bk := block.GetLatestBlock(st)
	bk2 := block.TestMakeNewBlockWithPrevBlock(bk, nil)

	si := &SyncInfo{
		Height: uint64(1),
		Block:  &bk2,
	}

	{
		err := v.Validate(ctx, si)
		require.NoError(t, err)
	}
}
