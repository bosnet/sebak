package runner

// import (
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/require"

// 	"boscoin.io/sebak/lib/ballot"
// 	"boscoin.io/sebak/lib/consensus"
// 	"boscoin.io/sebak/lib/consensus/round"
// 	"boscoin.io/sebak/lib/node"
// )

// 1. All 3 Nodes.
// 1. SequentialSelector.
// 1. When `ISAACStateManager` starts, the node proposes a ballot.
// 1. ISAACState is changed to `SIGN`.
// 1. TransitISAACState(ACCEPT) method is called.
// 1. ISAACState is changed to `ACCEPT`.
// 1. TimeoutACCEPT is a millisecond.
// 1. After timeout, ISAACState is back to `INIT`
// func TestStateTransitSequentialSelector(t *testing.T) {
// 	recv := make(chan struct{})
// 	conf := consensus.NewISAACConfiguration()
// 	conf.TimeoutINIT = time.Hour
// 	conf.TimeoutSIGN = time.Hour
// 	conf.TimeoutACCEPT = time.Hour

// 	nr, nodes, cm := createNodeRunnerForTesting(3, conf, recv)
// 	nr.Consensus().SetProposerSelector(consensus.NewSequentialSelector(nr.ConnectionManager()))
// 	nr.Consensus().SetLatestConfirmedBlock(genesisBlock)
// 	latestBlock := nr.Consensus().LatestConfirmedBlock()
// 	round := round.Round{
// 		Number:      0,
// 		BlockHeight: latestBlock.Height,
// 		BlockHash:   latestBlock.Hash,
// 		TotalTxs:    latestBlock.TotalTxs,
// 	}

// 	proposerAddress := nr.Consensus().SelectProposer(2, 0)
// 	proposer := getProposerNode(proposerAddress, nodes)
// 	require.NotNil(t, proposer)

// 	var err error
// 	t.Log("n0(Self)", "address", nr.localNode.Address())
// 	t.Log("n1", "address", nodes[1].Address())
// 	t.Log("n2", "address", nodes[2].Address())

// 	nr.StartStateManager()
// 	defer nr.StopStateManager()

// 	if proposerAddress == nr.localNode.Address() {
// 		t.Log("self is proposer")
// 		<-recv
// 		require.Equal(t, 1, len(cm.Messages()))
// 		cm.Flush()
// 	} else {
// 		t.Log("The other is proposer")
// 		ballotINIT := GenerateEmptyTxBallot(t, proposer, round, ballot.StateINIT, proposer)
// 		err = ReceiveBallot(t, nr, ballotINIT)
// 		require.Nil(t, err)
// 		<-recv

// 		require.Equal(t, 1, len(cm.Messages()))
// 		cm.Flush()
// 	}

// 	ballotSIGN1 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateSIGN, nodes[1])
// 	err = ReceiveBallot(t, nr, ballotSIGN1)
// 	require.Nil(t, err)

// 	ballotSIGN2 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateSIGN, nodes[2])
// 	err = ReceiveBallot(t, nr, ballotSIGN2)
// 	require.Nil(t, err)
// }

// func getProposerNode(proposer string, nodes []*node.LocalNode) *node.LocalNode {
// 	for _, node := range nodes {
// 		if node.Address() == proposer {
// 			return node
// 		}
// 	}
// 	return nil
// }
