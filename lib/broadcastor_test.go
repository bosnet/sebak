package sebak

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConectionManagerBroadcastor(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewISAACConfiguration()
	conf.TimeoutALLCONFIRM = 1 * time.Millisecond

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, len(b.Messages))
}
