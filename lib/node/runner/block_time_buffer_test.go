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
		1,
		5*time.Second,
		7*time.Second,
		3*time.Second,
		1*time.Second,
	))

	require.Equal(t, 2*time.Second, calculateBlockTimeBuffer(
		1,
		5*time.Second,
		3*time.Second,
		4*time.Second,
		1*time.Second,
	))

	require.Equal(t, time.Duration(0), calculateBlockTimeBuffer(
		1,
		5*time.Second,
		3*time.Second,
		7*time.Second,
		1*time.Second,
	))

	require.Equal(t, 2*time.Second, calculateBlockTimeBuffer(
		1,
		5*time.Second,
		5*time.Second,
		3*time.Second,
		1*time.Second,
	))
}

// We can check that after one year,
// if it took 6 sec to confirm one block,
// the next block should be confirmed about 4 seconds later.
func TestLongTermBlockTimeBuffer(t *testing.T) {
	oneYearHeight := uint64(365 * 24 * 60 * 60 / 5)
	oneYearSeconds := time.Duration(oneYearHeight) * 5 * time.Second

	oneYearHeight++
	oneYearSeconds += 6 * time.Second

	averageOneHeightAfter := oneYearSeconds / time.Duration(oneYearHeight)

	require.Equal(t, 4003462242*time.Nanosecond, calculateBlockTimeBuffer(
		oneYearHeight,
		5*time.Second,
		averageOneHeightAfter,
		0*time.Second,
		1*time.Second,
	))

	oneYearSeconds += 4003462242 * time.Nanosecond
	averageTwoHeightAfter := oneYearSeconds / time.Duration(oneYearHeight+1)
	require.Equal(t, 5*time.Second, averageTwoHeightAfter)
}
