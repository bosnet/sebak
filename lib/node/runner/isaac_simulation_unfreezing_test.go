/*
	In this file, there are unittests assume that one node receive a message from validators,
	and how the state of the node changes.
*/

package runner

import (
	"testing"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stretchr/testify/require"
)

/*
TestUnfreezingSimulation indicates the following:
	1. There are 3 nodes.
	2. The series of transaction are generated as follows, CreateAccount tx - Freezing tx - UnfreezingRequest tx - Unfreezing tx
	3. The node receives the SIGN, ACCEPT messages in order from the other two validator nodes.
	4. The node receives a ballot that exceeds the threshold, and the block is confirmed.
	5. Unfreezing tx will not be processed untill X period pass.
*/
func TestUnfreezingSimulation(t *testing.T) {
	nr, nodes, _ := createNodeRunnerForTesting(3, common.NewConfig(), nil)

	st := nr.storage

	proposer := nr.localNode

	// Generate create-account transaction
	tx, _, kpNewAccount := GetCreateAccountTransaction(uint64(0), uint64(500000000000))

	b1, _ := MakeConsensusAndBlock(t, tx, nr, nodes, proposer)
	require.Equal(t, b1.Height, uint64(2))

	// Generate create-frozen-account transaction
	tx2, _, kpFrozenAccount := GetFreezingTransaction(kpNewAccount, uint64(0), uint64(100000000000))

	b2, _ := MakeConsensusAndBlock(t, tx2, nr, nodes, proposer)

	ba, _ := block.GetBlockAccount(st, kpFrozenAccount.Address())

	require.Equal(t, b2.Height, uint64(3))
	require.Equal(t, uint64(ba.Balance), uint64(100000000000))

	// Generate unfreezing-request transaction
	tx3, _ := GetUnfreezingRequestTransaction(kpFrozenAccount, uint64(0))

	b3, _ := MakeConsensusAndBlock(t, tx3, nr, nodes, proposer)

	ba, _ = block.GetBlockAccount(st, kpFrozenAccount.Address())

	require.Equal(t, b3.Height, uint64(4))
	require.Equal(t, uint64(ba.Balance), uint64(99999990000))

	// Generate transaction for increasing blockheight
	tx4, _, _ := GetCreateAccountTransaction(uint64(1), uint64(1000000))

	b4, _ := MakeConsensusAndBlock(t, tx4, nr, nodes, proposer)
	require.Equal(t, b4.Height, uint64(5))

	// Generate transaction for increasing blockheight
	tx5, _, _ := GetCreateAccountTransaction(uint64(2), uint64(1000000))

	b5, _ := MakeConsensusAndBlock(t, tx5, nr, nodes, proposer)
	require.Equal(t, b5.Height, uint64(6))

	// Generate unfreezing-transaction not yet reached unfreezing blockheight
	tx6, _ := GetUnfreezingTransaction(kpFrozenAccount, kpNewAccount, uint64(1), uint64(99999980000))

	nr.TransactionPool.Add(tx6)
	roundNumber := uint64(0)
	_, err := nr.proposeNewBallot(roundNumber)
	require.NoError(t, err)

	require.False(t, nr.TransactionPool.Has(tx6.GetHash()))

	ba, _ = block.GetBlockAccount(st, kpFrozenAccount.Address())
	require.Equal(t, uint64(ba.Balance), uint64(99999990000))

	// Generate transaction for increasing blockheight
	tx7, _, _ := GetCreateAccountTransaction(uint64(3), uint64(1000000))

	b7, _ := MakeConsensusAndBlock(t, tx7, nr, nodes, proposer)
	require.Equal(t, b7.Height, uint64(7))

	tx8, _, _ := GetCreateAccountTransaction(uint64(4), uint64(1000000))

	b8, _ := MakeConsensusAndBlock(t, tx8, nr, nodes, proposer)
	require.Equal(t, b8.Height, uint64(8))

	tx9, _, _ := GetCreateAccountTransaction(uint64(5), uint64(1000000))

	b9, _ := MakeConsensusAndBlock(t, tx9, nr, nodes, proposer)
	require.Equal(t, b9.Height, uint64(9))

	// Generate unfreezing transaction
	tx10, _ := GetUnfreezingTransaction(kpFrozenAccount, kpNewAccount, uint64(1), uint64(99999980000))

	b10, _ := MakeConsensusAndBlock(t, tx10, nr, nodes, proposer)
	ba, _ = block.GetBlockAccount(st, kpFrozenAccount.Address())

	require.Equal(t, b10.Height, uint64(9))
	require.Equal(t, uint64(ba.Balance), uint64(99999990000))
}

func MakeConsensusAndBlock(t *testing.T, tx transaction.Transaction, nr *NodeRunner, nodes []*node.LocalNode, proposer *node.LocalNode) (block block.Block, err error) {

	nr.TransactionPool.Add(tx)

	// Generate proposed ballot in nodeRunner
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

	conf := common.NewConfig()

	// Check that the transaction is in RunningRounds

	ballotSIGN1 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[1], conf)
	err = ReceiveBallot(nr, ballotSIGN1)
	require.NoError(t, err)

	ballotSIGN2 := GenerateBallot(proposer, round, tx, ballot.StateSIGN, nodes[2], conf)
	err = ReceiveBallot(nr, ballotSIGN2)
	require.NoError(t, err)

	rr := nr.Consensus().RunningRounds[round.Index()]
	require.Equal(t, 2, len(rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)))

	ballotACCEPT1 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[1], conf)
	err = ReceiveBallot(nr, ballotACCEPT1)
	require.NoError(t, err)

	ballotACCEPT2 := GenerateBallot(proposer, round, tx, ballot.StateACCEPT, nodes[2], conf)
	err = ReceiveBallot(nr, ballotACCEPT2)

	require.Equal(t, 2, len(rr.Voted[proposer.Address()].GetResult(ballot.StateACCEPT)))

	block = nr.Consensus().LatestBlock()

	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	return
}
