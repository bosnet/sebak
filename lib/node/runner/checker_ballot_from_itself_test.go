// We can test that a node broadcast propose ballot or B(`EXP`) in ISAACStateManager.
package runner

import (
	"testing"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/consensus/round"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
)

func TestCheckerBallotFromItself(t *testing.T) {
	_, validTx := transaction.TestMakeTransaction(networkID, 1)

	nodeRunner, localNode := MakeNodeRunner()
	r := round.Round{
		Number:      0,
		BlockHeight: 1,
		BlockHash:   "",
		TotalTxs:    1,
		TotalOps:    2,
	}
	blt := ballot.NewBallot(localNode.Address(), localNode.Address(), r, []string{validTx.GetHash()})
	require.NotNil(t, blt)
	blt.Sign(localNode.Keypair(), networkID)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{},
		NodeRunner:     nodeRunner,
		LocalNode:      localNode,
		NetworkID:      networkID,
		Log:            nodeRunner.Log(),
		Ballot:         *blt,
	}

	err := BallotFromItself(checker)
	require.Error(t, err)
}
