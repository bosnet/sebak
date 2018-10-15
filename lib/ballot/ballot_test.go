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

/*
//	TestBallotHash checks that ballot.GetHash() makes non-empty hash.
func TestBallotHash(t *testing.T) {
	nodeRunners := createTestNodeRunner(1)

	nodeRunner := nodeRunners[0]

	round := round.Round{
		Number:      0,
		BlockHeight: nodeRunner.Consensus().LatestBlock.Height,
		BlockHash:   nodeRunner.Consensus().LatestBlock.Hash,
		TotalTxs:    nodeRunner.Consensus().LatestBlock.TotalTxs,
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

	conf := common.NewConfig()
	{
		ballot := NewBallot(node.Address(), node.Address(), round, []string{})

		opc, _ := NewCollectTxFeeFromBallot(*ballot, commonKP.Address())
		opi, _ := NewInflationFromBallot(*ballot, commonKP.Address(), common.Amount(1))
		ptx, _ := NewProposerTransactionFromBallot(*ballot, opc, opi)

		ballot.SetProposerTransaction(ptx)
		ballot.Sign(kp, networkID)

		err := ballot.IsWellFormed(networkID, conf)
		require.Nil(t, err)
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
		require.NotNil(t, err)
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
			require.Nil(t, err)
			ptx, err = NewProposerTransaction(kp.Address(), op)
			require.Nil(t, err)
		}

		blt.SetProposerTransaction(ptx)
		blt.Sign(node.Keypair(), networkID)
		err := blt.IsWellFormed(networkID, conf)
		require.NotNil(t, err)
	}
}

func TestBallotProposerAddress(t *testing.T) {
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
