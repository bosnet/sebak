package runner

import (
	"testing"

	"boscoin.io/sebak/lib/consensus"
	"github.com/stretchr/testify/require"
)

// In TestProposerCalculator test, the proposer is always the node itself because of SelfProposerCalculator.
func TestProposerCalculator(t *testing.T) {
	nodeRunners := createTestNodeRunner(1, consensus.NewISAACConfiguration())

	nodeRunner := nodeRunners[0]
	nodeRunner.SetProposerCalculator(SelfProposerCalculator{})

	require.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(1, 0))
	require.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(2, 0))
	require.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(2, 1))
}

// All 3 nodes have the same proposer at each round
func TestNodesHaveSameProposers(t *testing.T) {
	numberOfNodes := 3

	nodeRunners := createTestNodeRunner(numberOfNodes, consensus.NewISAACConfiguration())

	nr0 := nodeRunners[0]
	nr1 := nodeRunners[1]
	nr2 := nodeRunners[2]

	var maximumBlockHeight uint64 = 3
	var maximumRoundNumber uint64 = 3

	proposers0 := make([]string, maximumBlockHeight*maximumRoundNumber)
	proposers1 := make([]string, maximumBlockHeight*maximumRoundNumber)
	proposers2 := make([]string, maximumBlockHeight*maximumRoundNumber)

	for i := uint64(0); i < maximumBlockHeight; i++ {
		for j := uint64(0); j < maximumRoundNumber; j++ {
			proposers0[i*maximumRoundNumber] = nr0.CalculateProposer(i, j)
			proposers1[i*maximumRoundNumber] = nr1.CalculateProposer(i, j)
			proposers2[i*maximumRoundNumber] = nr2.CalculateProposer(i, j)
		}
	}

	require.Equal(t, proposers0, proposers1)
	require.Equal(t, proposers0, proposers2)
	require.Equal(t, proposers1, proposers2)
}
