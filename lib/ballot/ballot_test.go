package ballot

import (
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
)

func TestErrorBallotHasOverMaxTransactionsInBallot(t *testing.T) {
	kp := keypair.Random()
	commonKP := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	conf := common.NewTestConfig()
	conf.TxsLimit = 2
	_, tx := transaction.TestMakeTransaction(conf.NetworkID, 1)

	{
		blt := NewBallot(node.Address(), node.Address(), basis, []string{tx.GetHash()})

		opc, _ := NewCollectTxFeeFromBallot(*blt, commonKP.Address(), tx)
		opi, _ := NewInflationFromBallot(*blt, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*blt, opc, opi)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), conf.NetworkID)
		require.Nil(t, blt.IsWellFormed(conf))
	}

	{
		var txHashes []string
		var txs []transaction.Transaction
		for i := 0; i < conf.TxsLimit+1; i++ {
			_, tx := transaction.TestMakeTransaction(conf.NetworkID, 1)
			txs = append(txs, tx)
			txHashes = append(txHashes, tx.GetHash())
		}

		blt := NewBallot(node.Address(), node.Address(), basis, txHashes)

		opc, _ := NewCollectTxFeeFromBallot(*blt, commonKP.Address(), tx)
		opi, _ := NewInflationFromBallot(*blt, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*blt, opc, opi)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), conf.NetworkID)

		err := blt.IsWellFormed(conf)
		require.Error(t, err, errors.BallotHasOverMaxTransactionsInBallot)
	}
}

func TestBallotBadConfirmedTime(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	commonKP := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	basis := voting.Basis{Round: 0, Height: 0, BlockHash: "showme", TotalTxs: 0}

	updateBallot := func(ballot *Ballot) {
		ballot.H.Hash = ballot.B.MakeHashString()
		signature, _ := keypair.MakeSignature(kp, conf.NetworkID, ballot.H.Hash)
		ballot.H.Signature = base58.Encode(signature)
	}

	{
		ballot := NewBallot(node.Address(), node.Address(), basis, []string{})

		opc, _ := NewCollectTxFeeFromBallot(*ballot, commonKP.Address())
		opi, _ := NewInflationFromBallot(*ballot, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*ballot, opc, opi)

		ballot.SetProposerTransaction(ptx)
		ballot.Sign(kp, conf.NetworkID)

		err := ballot.IsWellFormed(conf)
		require.NoError(t, err)
	}

	{ // bad `Ballot.B.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), node.Address(), basis, []string{})
		ballot.Sign(kp, conf.NetworkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(conf)
		require.Error(t, err, errors.MessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), node.Address(), basis, []string{})
		ballot.Sign(kp, conf.NetworkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(conf)
		require.Error(t, err, errors.MessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), node.Address(), basis, []string{})
		ballot.Sign(kp, conf.NetworkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(conf)
		require.Error(t, err, errors.MessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), node.Address(), basis, []string{})
		ballot.Sign(kp, conf.NetworkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(conf)
		require.Error(t, err, errors.MessageHasIncorrectTime)
	}
}

func TestBallotEmptyHash(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	node, _ := node.NewLocalNode(kp, &common.Endpoint{}, "")
	r := voting.Basis{}
	b := NewBallot(node.Address(), node.Address(), r, []string{})
	b.Sign(kp, conf.NetworkID)

	require.True(t, len(b.GetHash()) > 0)
}

// TestBallotProposerTransaction checks; the proposed Ballot must have the
// proposed transaction.
func TestBallotProposerTransaction(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	commonKP := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	{ // without ProposerTransaction
		blt := NewBallot(node.Address(), node.Address(), basis, []string{})
		blt.Sign(node.Keypair(), conf.NetworkID)
		err := blt.IsWellFormed(conf)
		require.Error(t, err)
	}

	{ // with ProposerTransaction
		blt := NewBallot(node.Address(), node.Address(), basis, []string{})
		opb := operation.NewCollectTxFee(
			commonKP.Address(),
			common.Amount(10),
			uint64(len(blt.Transactions())),
			blt.VotingBasis().Height,
			blt.VotingBasis().BlockHash,
			blt.VotingBasis().TotalTxs,
		)
		var ptx ProposerTransaction
		{
			op, err := operation.NewOperation(opb)
			require.NoError(t, err)
			ptx, err = NewProposerTransaction(kp.Address(), op)
			require.NoError(t, err)
		}

		blt.SetProposerTransaction(ptx)
		blt.Sign(node.Keypair(), conf.NetworkID)
		err := blt.IsWellFormed(conf)
		require.Error(t, err)
	}
}

func TestNewBallot(t *testing.T) {
	kp := keypair.Random()
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")
	n, _ := node.NewLocalNode(kp, nodeEndpoint, "")
	p, _ := node.NewLocalNode(kp, proposerEndpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	b := NewBallot(n.Address(), p.Address(), basis, []string{})

	require.Equal(t, n.Address(), b.Source())
	require.Equal(t, p.Address(), b.Proposer())
}

// In this test, we can check that the normal ballot(not expired) should be signed by proposer.
func TestIsBallotWellFormed(t *testing.T) {
	conf := common.NewTestConfig()
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	tx := transaction.MakeTransactionCreateAccount(conf.NetworkID, nodeKP, keypair.Random().Address(), initialBalance)

	wellBallot := NewBallot(n.Address(), p.Address(), basis, []string{tx.GetHash()})

	commonKP := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)

	opi, _ := NewInflationFromBallot(*wellBallot, commonAccount.Address, initialBalance)
	opc, _ := NewCollectTxFeeFromBallot(*wellBallot, commonAccount.Address, tx)
	ptx, _ := NewProposerTransactionFromBallot(*wellBallot, opc, opi)
	wellBallot.SetProposerTransaction(ptx)

	wellBallot.Sign(proposerKP, conf.NetworkID)

	err := wellBallot.IsWellFormed(conf)

	require.NoError(t, err)

	wrongSignedBallot := NewBallot(n.Address(), p.Address(), basis, []string{tx.GetHash()})
	wrongSignedBallot.SetProposerTransaction(ptx)

	wrongSignedBallot.Sign(nodeKP, conf.NetworkID)

	err = wrongSignedBallot.IsWellFormed(conf)

	require.Error(t, err)

}

// We can check that expired ballot could be signed by node key pair(not proposer).
func TestIsExpiredBallotWellFormed(t *testing.T) {
	conf := common.NewTestConfig()
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	tx := transaction.MakeTransactionCreateAccount(conf.NetworkID, nodeKP, keypair.Random().Address(), initialBalance)

	b := NewBallot(n.Address(), p.Address(), basis, []string{tx.GetHash()})

	b.SetVote(StateSIGN, voting.EXP)
	b.Sign(nodeKP, conf.NetworkID)

	err := b.IsWellFormed(conf)

	require.NoError(t, err)

}

// This test is the same as TestIsExpiredBallotWellFormed except that the proposal transaction is added.
// As a result, in expired ballot's validation, it does not matter whether there is a proposal transaction or not.
func TestIsExpiredBallotWithProposerTransactionWellFormed(t *testing.T) {
	conf := common.NewTestConfig()
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	basis := voting.Basis{Round: 0, Height: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	tx := transaction.MakeTransactionCreateAccount(conf.NetworkID, nodeKP, keypair.Random().Address(), initialBalance)

	b := NewBallot(n.Address(), p.Address(), basis, []string{tx.GetHash()})

	commonKP := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)

	opi, _ := NewInflationFromBallot(*b, commonAccount.Address, initialBalance)
	opc, _ := NewCollectTxFeeFromBallot(*b, commonAccount.Address, tx)
	ptx, _ := NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)

	b.SetVote(StateSIGN, voting.EXP)
	b.Sign(nodeKP, conf.NetworkID)

	err := b.IsWellFormed(conf)

	require.NoError(t, err)

}
