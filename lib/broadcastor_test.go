package sebak

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionManagerBroadcastor(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	recv := make(chan struct{})

	b := NewTestBroadcastor(recv)
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewISAACConfiguration()
	conf.TimeoutALLCONFIRM = 1 * time.Millisecond

	nr.SetConf(conf)

	nr.StartStateManager()

	<-recv
	assert.Equal(t, 1, len(b.Messages))
}
