package sebak

import (
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
)

func TestErrorBallotHasOverMaxTransactionsInBallot(t *testing.T) {
	MaxTransactionsInBallotOrig := MaxTransactionsInBallot
	defer func() {
		MaxTransactionsInBallot = MaxTransactionsInBallotOrig
	}()

	MaxTransactionsInBallot = 2

	_, node := createNetMemoryNetwork()
	round := round.Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}
	_, tx := TestMakeTransaction(networkID, 1)

	ballot := NewBallot(node, round, []string{tx.GetHash()})
	ballot.Sign(node.Keypair(), networkID)
	require.Nil(t, ballot.IsWellFormed(networkID))

	var txs []string
	for i := 0; i < MaxTransactionsInBallot+1; i++ {
		_, tx := TestMakeTransaction(networkID, 1)
		txs = append(txs, tx.GetHash())
	}

	ballot = NewBallot(node, round, txs)
	ballot.Sign(node.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Error(t, err, errors.ErrorBallotHasOverMaxTransactionsInBallot)
}

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

func TestBallotBadConfirmedTime(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, _ := sebakcommon.NewEndpointFromString("https://localhost:1000")
	node, _ := sebaknode.NewLocalNode(kp, endpoint, "")

	round := round.Round{Number: 0, BlockHeight: 0, BlockHash: "", TotalTxs: 0}

	updateBallot := func(ballot *Ballot) {
		ballot.H.Hash = ballot.B.MakeHashString()
		signature, _ := sebakcommon.MakeSignature(kp, networkID, ballot.H.Hash)
		ballot.H.Signature = base58.Encode(signature)
	}

	{
		ballot := NewBallot(node, round, []string{})
		ballot.Sign(kp, networkID)

		err := ballot.IsWellFormed(networkID)
		require.Nil(t, err)
	}

	{ // bad `Ballot.B.Confirmed` time; too ahead
		ballot := NewBallot(node, round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = sebakcommon.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Confirmed` time; too behind
		ballot := NewBallot(node, round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * BallotConfirmedTimeAllowDuration)
		ballot.B.Confirmed = sebakcommon.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too ahead
		ballot := NewBallot(node, round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(2) * BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = sebakcommon.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}

	{ // bad `Ballot.B.Proposed.Confirmed` time; too behind
		ballot := NewBallot(node, round, []string{})
		ballot.Sign(kp, networkID)

		newConfirmed := time.Now().Add(time.Duration(-2) * BallotConfirmedTimeAllowDuration)
		ballot.B.Proposed.Confirmed = sebakcommon.FormatISO8601(newConfirmed)
		updateBallot(ballot)

		err := ballot.IsWellFormed(networkID)
		require.Error(t, err, errors.ErrorMessageHasIncorrectTime)
	}
}
