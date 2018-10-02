package sync

import (
	"context"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	networkID := []byte("test-network")
	_, nw, _ := network.CreateMemoryNetwork(nil)

	commonCfg := common.Config{}

	v := NewBlockValidator(nw, st, networkID, commonCfg)

	ctx := context.Background()

	bk := block.TestMakeNewBlock(nil)
	err := bk.Save(st)
	require.Nil(t, err)

	bk2 := block.TestMakeNewBlockWithPrevBlock(bk, nil)

	si := &SyncInfo{
		BlockHeight: uint64(1),
		Block:       &bk2,
	}

	{
		err := v.Validate(ctx, si)
		require.Nil(t, err)
	}
}
