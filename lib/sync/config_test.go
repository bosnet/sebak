package sync

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	conf := common.NewTestConfig()
	st := block.InitTestBlockchain()
	node := node.NewTestLocalNode0()
	cm := &mockConnectionManager{}
	tp := transaction.NewPool(conf)

	cfg, err := NewConfig(node, st, cm, tp, conf)
	require.NoError(t, err)
	cfg.SyncPoolSize = 100
	cfg.logger = common.NopLogger()

	syncer := cfg.NewSyncer()

	require.NotNil(t, syncer)
	require.Equal(t, syncer.poolSize, cfg.SyncPoolSize)
}
