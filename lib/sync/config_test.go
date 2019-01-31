package sync

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	conf := common.NewTestConfig()
	st := block.InitTestBlockchain()
	_, nt, _ := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}
	tp := transaction.NewPool(conf)

	endpoint := common.MustParseEndpoint("https://localhost:5000?NodeName=n1")
	node, _ := node.NewLocalNode(keypair.Random(), endpoint, "")

	cfg, err := NewConfig(node, st, nt, cm, tp, conf)
	require.NoError(t, err)
	cfg.SyncPoolSize = 100
	cfg.logger = common.NopLogger()

	syncer := cfg.NewSyncer()

	require.NotNil(t, syncer)
	require.Equal(t, syncer.poolSize, cfg.SyncPoolSize)
}
