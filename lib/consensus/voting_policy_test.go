package consensus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThreshold(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(66)
	require.NoError(t, err)

	vt.SetValidators(1000)
	require.Equal(t, 660, vt.Threshold())

}
