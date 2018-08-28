package sebak

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type SelfProposerCalculator struct {
}

func (c SelfProposerCalculator) Calculate(nr *NodeRunner, _ uint64, _ uint64) string {
	return nr.localNode.Address()
}

func TestProposerCalculator(t *testing.T) {
	nodeRunners := createTestNodeRunner(1)

	nodeRunner := nodeRunners[0]
	nodeRunner.SetProposerCalculator(SelfProposerCalculator{})

	assert.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(0, 1))
	assert.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(0, 2))
	assert.Equal(t, nodeRunner.localNode.Address(), nodeRunner.CalculateProposer(1, 2))
}
