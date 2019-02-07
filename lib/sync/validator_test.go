package sync

import (
	"context"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	conf := common.NewTestConfig()
	st := block.InitTestBlockchain()
	defer st.Close()
	nw, _ := network.CreateMemoryNetwork(nil)
	tp := transaction.NewPool(conf)

	v := NewBlockValidator(nw, st, tp, conf)

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
