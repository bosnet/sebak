package sebak

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/error"
)

func TestErrorBallotHasOverMaxTransactionsInBallot(t *testing.T) {
	MaxTransactionsInBallotOrig := MaxTransactionsInBallot
	defer func() {
		MaxTransactionsInBallot = MaxTransactionsInBallotOrig
	}()

	MaxTransactionsInBallot = 2

	_, node := createNetMemoryNetwork()
	round := Round{Number: 0, BlockHeight: 1, BlockHash: "hahaha", TotalTxs: 1}
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
	require.Error(t, err, sebakerror.ErrorBallotHasOverMaxTransactionsInBallot)
}
