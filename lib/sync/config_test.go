package sync

import (
	"fmt"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()
	_, nt, _ := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}

	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	node, _ := node.NewLocalNode(keypair.Random(), endpoint, "")

	cfg := NewConfig(conf.NetworkID, node, st, nt, cm, conf)
	cfg.SyncPoolSize = 100
	cfg.logger = common.NopLogger()

	syncer := cfg.NewSyncer()

	require.NotNil(t, syncer)
	require.Equal(t, syncer.poolSize, cfg.SyncPoolSize)
}
