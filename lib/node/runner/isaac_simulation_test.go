/*
	In this file, there are unittests assume that one node receive a message from validators,
	and how the state of the node changes.
*/

package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
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
	nr, nodes, _ := createNodeRunnerForTesting(5, consensus.NewISAACConfiguration(), nil)
	tx, txByte := GetTransaction(t)

	message := common.NetworkMessage{Type: common.TransactionMessage, Data: txByte}

	// `nr` is proposer's runner
	proposer := nr.localNode

	nr.Consensus().SetLatestConfirmedBlock(genesisBlock)

	var err error
	err = nr.handleTransaction(message)

	require.Nil(t, err)
	require.True(t, nr.Consensus().TransactionPool.Has(tx.GetHash()))

	// Generate proposed ballot in nr
	roundNumber := uint64(0)
	err = nr.proposeNewBallot(roundNumber)
	require.Nil(t, err)

	b := nr.Consensus().LatestConfirmedBlock()
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: b.Height,
		BlockHash:   b.Hash,
		TotalTxs:    b.TotalTxs,
	}
	runningRounds := nr.Consensus().RunningRounds

	// Check that the transaction is in RunningRounds
	rr := runningRounds[round.BlockHeight]
	txHashs := rr.Transactions[proposer.Address()]
	require.Equal(t, tx.GetHash(), txHashs[0])

	ballotSIGN1 := GenerateBallot(t, proposer, round, tx, ballot.StateSIGN, nodes[1])
	err = ReceiveBallot(t, nr, ballotSIGN1)
	require.Nil(t, err)

	ballotSIGN2 := GenerateBallot(t, proposer, round, tx, ballot.StateSIGN, nodes[2])
	err = ReceiveBallot(t, nr, ballotSIGN2)
	require.Nil(t, err)

	ballotSIGN3 := GenerateBallot(t, proposer, round, tx, ballot.StateSIGN, nodes[3])
	err = ReceiveBallot(t, nr, ballotSIGN3)
	require.Nil(t, err)

	ballotSIGN4 := GenerateBallot(t, proposer, round, tx, ballot.StateSIGN, nodes[4])
	err = ReceiveBallot(t, nr, ballotSIGN4)
	require.Nil(t, err)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)))

	ballotACCEPT1 := GenerateBallot(t, proposer, round, tx, ballot.StateACCEPT, nodes[1])
	err = ReceiveBallot(t, nr, ballotACCEPT1)
	require.Nil(t, err)

	ballotACCEPT2 := GenerateBallot(t, proposer, round, tx, ballot.StateACCEPT, nodes[2])
	err = ReceiveBallot(t, nr, ballotACCEPT2)
	require.Nil(t, err)

	ballotACCEPT3 := GenerateBallot(t, proposer, round, tx, ballot.StateACCEPT, nodes[3])
	err = ReceiveBallot(t, nr, ballotACCEPT3)
	require.Nil(t, err)

	ballotACCEPT4 := GenerateBallot(t, proposer, round, tx, ballot.StateACCEPT, nodes[4])
	err = ReceiveBallot(t, nr, ballotACCEPT4)

	_, ok := err.(CheckerStopCloseConsensus)
	require.True(t, ok)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateACCEPT)))

	block := nr.Consensus().LatestConfirmedBlock()
	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	require.Equal(t, tx.GetHash(), block.Transactions[0])
}
