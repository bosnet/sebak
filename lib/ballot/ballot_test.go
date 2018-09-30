package ballot

import (
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-test-network")

func TestErrorBallotHasOverMaxTransactionsInBallot(t *testing.T) {
	MaxTransactionsInBallotOrig := common.MaxTransactionsInBallot
	defer func() {
		common.MaxTransactionsInBallot = MaxTransactionsInBallotOrig
	}()

	common.MaxTransactionsInBallot = 2

	kp, _ := keypair.Random()
	commonKP, _ := keypair.Random()
	endpoint, _ := common.NewEndpointFromString("https://localhost:1000")
	node, _ := node.NewLocalNode(kp, endpoint, "")

	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}
	_, tx := transaction.TestMakeTransaction(networkID, 1)

	{
		blt := NewBallot(node.Address(), round, []string{tx.GetHash()})

		ptx, _ := NewProposerTransactionFromBallot(*blt, commonKP.Address(), tx)
		ptx.Sign(node.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), networkID)
		require.Nil(t, blt.IsWellFormed(networkID))
	}

	{
		var txHashes []string
		var txs []transaction.Transaction
		for i := 0; i < common.MaxTransactionsInBallot+1; i++ {
			_, tx := transaction.TestMakeTransaction(networkID, 1)
			txs = append(txs, tx)
			txHashes = append(txHashes, tx.GetHash())
		}

		blt := NewBallot(node.Address(), round, txHashes)

		ptx, _ := NewProposerTransactionFromBallot(*blt, commonKP.Address(), txs...)
		ptx.Sign(node.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)

		blt.Sign(node.Keypair(), networkID)

		err := blt.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorBallotHasOverMaxTransactionsInBallot)
	}
}

/*
//	TestBallotHash checks that ballot.GetHash() makes non-empty hash.
func TestBallotHash(t *testing.T) {
	nodeRunners := createTestNodeRunner(1)

	nodeRunner := nodeRunners[0]

	round := round.Round{
		Number:      0,
		BlockHeight: nodeRunner.Consensus().LatestConfirmedBlock.Height,
		BlockHash:   nodeRunner.Consensus().LatestConfirmedBlock.Hash,
		TotalTxs:    nodeRunner.Consensus().LatestConfirmedBlock.TotalTxs,
	}

	ballot := NewBallot(nodeRunner.localNode, round, []string{})
	ballot.Sign(kp, networkID)
	require.NotZero(t, len(ballot.GetHash()))

}
*/

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

	{
		ballot := NewBallot(node.Address(), round, []string{})
		ballot := NewBallot(node.Address(), round, []string{})
		ptx, _ := NewProposerTransactionFromBallot(*ballot, commonKP.Address())
		ptx.Sign(kp, networkID)
		ballot.SetProposerTransaction(ptx)
		ballot.Sign(kp, networkID)

		err := ballot.IsWellFormed(networkID)
		require.Nil(t, err)
	}

	{ // bad `Ballot.B.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too ahead
		ballot := NewBallot(node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too behind
		ballot := NewBallot(node.Address(), round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * common.BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = common.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}
}

func TestBallotEmptyHash(t *testing.T) {
	kp, _ := keypair.Random()
	node, _ := node.NewLocalNode(kp, &common.Endpoint{}, "")
	r := round.Round{}
	b := NewBallot(node.Address(), r, []string{})
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

	{ // without ProposerTransaction
		blt := NewBallot(node, round, []string{})
		blt.Sign(node.Keypair(), networkID)
		err := blt.IsWellFormed(networkID)
		require.NotNil(t, err)
	}

	{ // with ProposerTransaction
		blt := NewBallot(node, round, []string{})
		opb := transaction.NewOperationBodyCollectTxFee(
			commonKP.Address(),
			common.Amount(10),
			uint64(len(blt.Transactions())),
			blt.Round().BlockHeight,
			blt.Round().BlockHash,
			blt.Round().TotalTxs,
		)
		var ptx ProposerTransaction
		{
			op, err := transaction.NewOperation(opb)
			require.Nil(t, err)
			ptx, err = NewProposerTransaction(kp.Address(), op)
			require.Nil(t, err)
		}

		blt.SetProposerTransaction(ptx)
		blt.Sign(node.Keypair(), networkID)
		err := blt.IsWellFormed(networkID)
		require.NotNil(t, err)
	}
}
