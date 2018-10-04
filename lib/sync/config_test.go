package sync

import (
	"fmt"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	st := storage.NewTestStorage()
	_, nt, _ := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}

	kp, _ := keypair.Random()
	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	node, _ := node.NewLocalNode(kp, endpoint, "")
	networkID := []byte("test-network")

	cfg := NewConfig(networkID, node, st, nt, cm)
	cfg.SyncPoolSize = 100
	cfg.logger = NopLogger()

	syncer := cfg.NewSyncer()

	require.NotNil(t, syncer)
	require.Equal(t, syncer.poolSize, cfg.SyncPoolSize)
}
