package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculateAverageBlockTime(t *testing.T) {
	now := time.Now()
	lastDay := now.AddDate(0, 0, -1)

	nBlocksInOneDay := 720 * 24
	height := uint64(nBlocksInOneDay)

	blockTime := calculateAverageBlockTime(lastDay, height)
	require.True(t, blockTime > 4900*time.Millisecond)
	require.True(t, blockTime < 5100*time.Millisecond)

}

func TestCalculateBlockTimeBuffer(t *testing.T) {
	require.Equal(t, 1*time.Second, calculateBlockTimeBuffer(
		5*time.Second,
		7*time.Second,
		3*time.Second,
		1*time.Second,
	))

	require.Equal(t, 2*time.Second, calculateBlockTimeBuffer(
		5*time.Second,
		3*time.Second,
		4*time.Second,
		1*time.Second,
	))

	require.Equal(t, time.Duration(0), calculateBlockTimeBuffer(
		5*time.Second,
		3*time.Second,
		7*time.Second,
		1*time.Second,
	))

	require.Equal(t, 2*time.Second, calculateBlockTimeBuffer(
		5*time.Second,
		5020*time.Millisecond,
		3*time.Second,
		1*time.Second,
	))
}
