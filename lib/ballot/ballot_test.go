package ballot

import (
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

var networkID []byte = []byte("sebak-test-network")

func TestErrorBallotHasOverMaxTransactionsInBallot(t *testing.T) {

	kp, _ := keypair.Random()
	commonKP, _ := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	_, tx := transaction.TestMakeTransaction(networkID, 1)
	conf := common.NewConfig()
	conf.TxsLimit = 2

	{
		blt := NewBallot(node.Address(), node.Address(), round, []string{tx.GetHash()})

		opc, _ := NewCollectTxFeeFromBallot(*blt, commonKP.Address(), tx)
		opi, _ := NewInflationFromBallot(*blt, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*blt, opc, opi)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), networkID)
		require.Nil(t, blt.IsWellFormed(networkID, conf))
	}

	{
		var txHashes []string
		var txs []transaction.Transaction
		for i := 0; i < conf.TxsLimit+1; i++ {
			_, tx := transaction.TestMakeTransaction(networkID, 1)
			txs = append(txs, tx)
			txHashes = append(txHashes, tx.GetHash())
		}

		blt := NewBallot(node.Address(), node.Address(), round, txHashes)

		opc, _ := NewCollectTxFeeFromBallot(*blt, commonKP.Address(), tx)
		opi, _ := NewInflationFromBallot(*blt, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*blt, opc, opi)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), networkID)

		err := blt.IsWellFormed(networkID, conf)
		require.Error(t, err, errors.ErrorBallotHasOverMaxTransactionsInBallot)
	}
}

func TestBallotBadConfirmedTime(t *testing.T) {
	kp, _ := keypair.Random()
	commonKP, _ := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	round := round.Round{Number: 0, BlockHeight: 0, BlockHash: "showme", TotalTxs: 0}

	updateBallot := func(ballot *Ballot) {
		ballot.H.Hash = ballot.B.MakeHashString()
		signature, _ := common.MakeSignature(kp, networkID, ballot.H.Hash)
		ballot.H.Signature = base58.Encode(signature)
	}

	conf := common.NewConfig()
	{
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})

		opc, _ := NewCollectTxFeeFromBallot(*ballot, commonKP.Address())
		opi, _ := NewInflationFromBallot(*ballot, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*ballot, opc, opi)

		ballot.SetProposerTransaction(ptx)
		ballot.Sign(kp, networkID)

		err := ballot.IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{ // bad `Ballot.B.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID, conf)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID, conf)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID, conf)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID, conf)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}
}

func TestBallotEmptyHash(t *testing.T) {
	kp, _ := keypair.Random()
	node, _ := node.NewLocalNode(kp, &common.Endpoint{}, "")
	r := round.Round{}
	b := NewBallot(node.Address(), node.Address(), r, []string{})
	b.Sign(kp, networkID)

	require.True(t, len(b.GetHash()) > 0)
}

// TestBallotProposerTransaction checks; the proposed Ballot must have the
// proposed transaction.
func TestBallotProposerTransaction(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	commonKP, _ := keypair.Random()

	conf := common.NewConfig()
	{ // without ProposerTransaction
		blt := NewBallot(node.Address(), node.Address(), round, []string{})
		blt.Sign(node.Keypair(), networkID)
		err := blt.IsWellFormed(networkID, conf)
		require.Error(t, err)
	}

	{ // with ProposerTransaction
		blt := NewBallot(node.Address(), node.Address(), round, []string{})
		opb := operation.NewCollectTxFee(
			commonKP.Address(),
			common.Amount(10),
			uint64(len(blt.Transactions())),
			blt.Round().BlockHeight,
			blt.Round().BlockHash,
			blt.Round().TotalTxs,
		)
		var ptx ProposerTransaction
		{
			op, err := operation.NewOperation(opb)
			require.NoError(t, err)
			ptx, err = NewProposerTransaction(kp.Address(), op)
			require.NoError(t, err)
		}

		blt.SetProposerTransaction(ptx)
		blt.Sign(node.Keypair(), networkID)
		err := blt.IsWellFormed(networkID, conf)
		require.Error(t, err)
	}
}

func TestNewBallot(t *testing.T) {
	kp, _ := keypair.Random()
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")
	n, _ := node.NewLocalNode(kp, nodeEndpoint, "")
	p, _ := node.NewLocalNode(kp, proposerEndpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	b := NewBallot(n.Address(), p.Address(), round, []string{})

	require.Equal(t, n.Address(), b.Source())
	require.Equal(t, p.Address(), b.Proposer())
}

// In this test, we can check that the normal ballot(not expired) should be signed by proposer.
func TestIsBallotWellFormed(t *testing.T) {
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP, _ := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP, _ := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	kpNewAccount, _ := keypair.Random()
	tx := transaction.MakeTransactionCreateAccount(nodeKP, kpNewAccount.Address(), initialBalance)

	wellBallot := NewBallot(n.Address(), p.Address(), round, []string{tx.GetHash()})

	commonKP, _ := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)

	opi, _ := NewInflationFromBallot(*wellBallot, commonAccount.Address, initialBalance)
	opc, _ := NewCollectTxFeeFromBallot(*wellBallot, commonAccount.Address, tx)
	ptx, _ := NewProposerTransactionFromBallot(*wellBallot, opc, opi)
	wellBallot.SetProposerTransaction(ptx)

	wellBallot.Sign(proposerKP, networkID)

	err := wellBallot.IsWellFormed(networkID, common.NewConfig())

	require.NoError(t, err)

	wrongSignedBallot := NewBallot(n.Address(), p.Address(), round, []string{tx.GetHash()})
	wrongSignedBallot.SetProposerTransaction(ptx)

	wrongSignedBallot.Sign(nodeKP, networkID)

	err = wrongSignedBallot.IsWellFormed(networkID, common.NewConfig())

	require.Error(t, err)

}

// We can check that expired ballot could be signed by node key pair(not proposer).
func TestIsExpiredBallotWellFormed(t *testing.T) {
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP, _ := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP, _ := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	kpNewAccount, _ := keypair.Random()
	tx := transaction.MakeTransactionCreateAccount(nodeKP, kpNewAccount.Address(), initialBalance)

	b := NewBallot(n.Address(), p.Address(), round, []string{tx.GetHash()})

	b.SetVote(StateSIGN, VotingEXP)
	b.Sign(nodeKP, networkID)

	err := b.IsWellFormed(networkID, common.NewConfig())

	require.NoError(t, err)

}

// This test is the same as TestIsExpiredBallotWellFormed except that the proposal transaction is added.
// As a result, in expired ballot's validation, it does not matter whether there is a proposal transaction or not.
func TestIsExpiredBallotWithProposerTransactionWellFormed(t *testing.T) {
	nodeEndpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	proposerEndpoint, _ := common.NewEndpointFromString("https://localhost:1001")

	nodeKP, _ := keypair.Random()
	n, _ := node.NewLocalNode(nodeKP, nodeEndpoint, "")

	proposerKP, _ := keypair.Random()
	p, _ := node.NewLocalNode(proposerKP, proposerEndpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}

	initialBalance := common.Amount(common.BaseReserve)
	kpNewAccount, _ := keypair.Random()
	tx := transaction.MakeTransactionCreateAccount(nodeKP, kpNewAccount.Address(), initialBalance)

	b := NewBallot(n.Address(), p.Address(), round, []string{tx.GetHash()})

	commonKP, _ := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)

	opi, _ := NewInflationFromBallot(*b, commonAccount.Address, initialBalance)
	opc, _ := NewCollectTxFeeFromBallot(*b, commonAccount.Address, tx)
	ptx, _ := NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)

	b.SetVote(StateSIGN, VotingEXP)
	b.Sign(nodeKP, networkID)

	err := b.IsWellFormed(networkID, common.NewConfig())

	require.NoError(t, err)

}
