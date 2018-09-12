/*
	In this file, there are unittests assume that one node receive a message from validators,
	and how the state of the node changes.
*/

package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
)

/*
TestISAACSimulationProposer indicates the following:
	1. Proceed for one round.
	2. The node is the proposer of this round.
	3. There are 5 nodes and threshold is 4.
	3. The node receives the SIGN, ACCEPT messages in order from the other four validator nodes.
	4. The node receives a ballot that exceeds the threshold, and the block is confirmed.
*/
func TestISAACSimulationProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(5, consensus.NewISAACConfiguration())
	tx, txByte := GetTransaction(t)

	message := common.NetworkMessage{Type: common.TransactionMessage, Data: txByte}

	nodeRunner := nodeRunners[0]

	// `nodeRunner` is proposer's runner
	nodeRunner.SetProposerCalculator(SelfProposerCalculator{})
	proposer := nodeRunner.localNode

	nodeRunner.Consensus().SetLatestConsensusedBlock(genesisBlock)

	var err error
	err = nodeRunner.handleTransaction(message)

	require.Nil(t, err)
	require.True(t, nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()))

	// Generate proposed ballot in nodeRunner
	roundNumber := uint64(0)
	err = nodeRunner.proposeNewBallot(roundNumber)
	require.Nil(t, err)
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: nodeRunner.Consensus().LatestConfirmedBlock.Height,
		BlockHash:   nodeRunner.Consensus().LatestConfirmedBlock.Hash,
		TotalTxs:    nodeRunner.Consensus().LatestConfirmedBlock.TotalTxs,
	}
	runningRounds := nodeRunner.Consensus().RunningRounds

	// Check that the transaction is in RunningRounds
	rr := runningRounds[round.Hash()]
	txHashs := rr.Transactions[proposer.Address()]
	require.Equal(t, tx.GetHash(), txHashs[0])

	ballotSIGN1 := GenerateBallot(t, proposer, round, tx, common.BallotStateSIGN, nodeRunners[1].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotSIGN1)
	require.Nil(t, err)

	ballotSIGN2 := GenerateBallot(t, proposer, round, tx, common.BallotStateSIGN, nodeRunners[2].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotSIGN2)
	require.Nil(t, err)

	ballotSIGN3 := GenerateBallot(t, proposer, round, tx, common.BallotStateSIGN, nodeRunners[3].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotSIGN3)
	require.Nil(t, err)

	ballotSIGN4 := GenerateBallot(t, proposer, round, tx, common.BallotStateSIGN, nodeRunners[4].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotSIGN4)
	require.Nil(t, err)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(common.BallotStateSIGN)))

	ballotACCEPT1 := GenerateBallot(t, proposer, round, tx, common.BallotStateACCEPT, nodeRunners[1].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotACCEPT1)
	require.Nil(t, err)

	ballotACCEPT2 := GenerateBallot(t, proposer, round, tx, common.BallotStateACCEPT, nodeRunners[2].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotACCEPT2)
	require.Nil(t, err)

	ballotACCEPT3 := GenerateBallot(t, proposer, round, tx, common.BallotStateACCEPT, nodeRunners[3].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotACCEPT3)
	require.Nil(t, err)

	ballotACCEPT4 := GenerateBallot(t, proposer, round, tx, common.BallotStateACCEPT, nodeRunners[4].localNode)
	err = ReceiveBallot(t, nodeRunner, ballotACCEPT4)

	_, ok := err.(CheckerStopCloseConsensus)
	require.True(t, ok)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(common.BallotStateACCEPT)))

	block := nodeRunner.Consensus().LatestConfirmedBlock
	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	require.Equal(t, tx.GetHash(), block.Transactions[0])
}
