/*
	In this file, there are unittests assume that one node receive a message from validators,
	and how the state of the node changes.
*/

package runner

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
)

func makeIssuancePublicFinanceAtOnce() *Issuance {
	hash := "public-finance-at-once" // originally it is the transaction hash
	start := uint64(123)
	end := uint64(123)
	interval := uint64(1)
	unit := common.Amount(10000)
	total := common.Amount(10000)
	kp := keypair.Master(hash)
	address := kp.Address()
	return NewIssuance(hash, start, end, interval, unit, total, address)
}

func makeIssuancePublicFinancePeriodic() *Issuance {
	hash := "public-finance-periodic" // originally it is the transaction hash
	start := uint64(103)
	end := uint64(300)
	interval := uint64(20)
	unit := common.Amount(10000)
	total := common.Amount(0)
	kp := keypair.Master(hash)
	address := kp.Address()
	return NewIssuance(hash, start, end, interval, unit, total, address)
}

func TestIssuanceTimingnTransactions(t *testing.T) {
	nrs := createTestNodeRunner(1)
	ip := nrs[0].IssuancePolicy()
	ip.Add(*makeIssuancePublicFinanceAtOnce())
	ip.Add(*makeIssuancePublicFinancePeriodic())
	kp, _ := keypair.Random()

	for bh := 1; bh < 301; bh++ {
		tx, avail := ip.Issue(uint64(bh), kp)
		tx.Sign(kp, nrs[0].NetworkID())
		require.True(t, avail)
		if avail {
			err := tx.IsWellFormed(nrs[0].NetworkID())
			require.Nil(t, err)
			if bh == 123 {
				require.Equal(t, 3, len(tx.B.Operations))
			} else if bh == 103 {
				require.Equal(t, 2, len(tx.B.Operations))
			} else if bh > 103 && (bh-103)%20 == 0 {
				require.Equal(t, 2, len(tx.B.Operations))
			} else {
				require.Equal(t, 1, len(tx.B.Operations))
			}
		}
	}
}

func TestIssuance(t *testing.T) {
	nodeRunners := createTestNodeRunner(5)
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
	require.Equal(t, 2, nodeRunner.Consensus().TransactionPool.Len())
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
