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
	nr, nodes, _ := createNodeRunnerForTesting(5, common.NewConfig(), nil)
	tx, _ := GetTransaction()

	// `nr` is proposer's runner
	proposer := nr.localNode

	var err error
	nr.TransactionPool.Add(tx)

	// Generate proposed ballot in nr
	roundNumber := uint64(0)
	_, err = nr.proposeNewBallot(roundNumber)
	require.NoError(t, err)

	b := nr.Consensus().LatestBlock()
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: b.Height,
		BlockHash:   b.Hash,
		TotalTxs:    b.TotalTxs,
	}
	require.True(t, nr.TransactionPool.Has(tx.GetHash()))

	conf := common.NewConfig()

	ballotSIGN1 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[1], conf)
	err = ReceiveBallot(nr, ballotSIGN1)
	require.NoError(t, err)

	ballotSIGN2 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[2], conf)
	err = ReceiveBallot(nr, ballotSIGN2)
	require.NoError(t, err)

	ballotSIGN3 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[3], conf)
	err = ReceiveBallot(nr, ballotSIGN3)
	require.NoError(t, err)

	ballotSIGN4 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[4], conf)
	err = ReceiveBallot(nr, ballotSIGN4)
	require.NoError(t, err)

	rr := nr.Consensus().RunningRounds[round.Index()]
	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)))

	ballotACCEPT0 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[0], conf)
	err = ReceiveBallot(nr, ballotACCEPT0)
	require.NoError(t, err)

	ballotACCEPT1 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[1], conf)
	err = ReceiveBallot(nr, ballotACCEPT1)
	require.NoError(t, err)

	ballotACCEPT2 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[2], conf)
	err = ReceiveBallot(nr, ballotACCEPT2)
	require.NoError(t, err)

	ballotACCEPT3 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[3], conf)
	err = ReceiveBallot(nr, ballotACCEPT3)

	_, ok := err.(CheckerStopCloseConsensus)
	require.True(t, ok)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateACCEPT)))

	block := nr.Consensus().LatestBlock()
	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	require.Equal(t, tx.GetHash(), block.Transactions[0])
}
