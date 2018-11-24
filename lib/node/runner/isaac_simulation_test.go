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
	"boscoin.io/sebak/lib/voting"
)

/*
TestISAACSimulationProposer indicates the following:
	1. Proceed for one votingBasis.
	2. The node is the proposer of this round.
	3. There are 5 nodes and threshold is 4.
	3. The node receives the SIGN, ACCEPT messages in order from the other four validator nodes.
	4. The node receives a ballot that exceeds the threshold, and the block is confirmed.
*/
func TestISAACSimulationProposer(t *testing.T) {
	conf := common.NewTestConfig()
	nr, nodes, _ := createNodeRunnerForTesting(5, conf, nil)
	tx, _ := GetTransaction()

	// `nr` is proposer's runner
	proposer := nr.localNode

	var err error
	nr.TransactionPool.Add(tx)

	// Generate proposed ballot in nr
	round := uint64(0)
	_, err = nr.proposeNewBallot(round)
	require.NoError(t, err)

	b := nr.Consensus().LatestBlock()
	votingBasis := voting.Basis{
		Round:     round,
		Height:    b.Height,
		BlockHash: b.Hash,
		TotalTxs:  b.TotalTxs,
	}
	require.True(t, nr.TransactionPool.Has(tx.GetHash()))

	ballotSIGN1 := GenerateBallot(proposer, votingBasis, tx, ballot.StateSIGN, nodes[1], conf)
	err = ReceiveBallot(nr, ballotSIGN1)
	require.NoError(t, err)

	ballotSIGN2 := GenerateBallot(proposer, votingBasis, tx, ballot.StateSIGN, nodes[2], conf)
	err = ReceiveBallot(nr, ballotSIGN2)
	require.NoError(t, err)

	ballotSIGN3 := GenerateBallot(proposer, votingBasis, tx, ballot.StateSIGN, nodes[3], conf)
	err = ReceiveBallot(nr, ballotSIGN3)
	require.NoError(t, err)

	ballotSIGN4 := GenerateBallot(proposer, votingBasis, tx, ballot.StateSIGN, nodes[4], conf)
	err = ReceiveBallot(nr, ballotSIGN4)
	require.NoError(t, err)

	rr := nr.Consensus().RunningRounds[votingBasis.Index()]
	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)))

	ballotACCEPT0 := GenerateBallot(proposer, votingBasis, tx, ballot.StateACCEPT, nodes[0], conf)
	err = ReceiveBallot(nr, ballotACCEPT0)
	require.NoError(t, err)

	ballotACCEPT1 := GenerateBallot(proposer, votingBasis, tx, ballot.StateACCEPT, nodes[1], conf)
	err = ReceiveBallot(nr, ballotACCEPT1)
	require.NoError(t, err)

	ballotACCEPT2 := GenerateBallot(proposer, votingBasis, tx, ballot.StateACCEPT, nodes[2], conf)
	err = ReceiveBallot(nr, ballotACCEPT2)
	require.NoError(t, err)

	ballotACCEPT3 := GenerateBallot(proposer, votingBasis, tx, ballot.StateACCEPT, nodes[3], conf)
	err = ReceiveBallot(nr, ballotACCEPT3)
	require.NoError(t, err)

	block := nr.Consensus().LatestBlock()
	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	require.Equal(t, tx.GetHash(), block.Transactions[0])
}
