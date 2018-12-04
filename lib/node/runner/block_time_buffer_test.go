package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
// the next block should be confirmed 4 seconds later.
func TestLongTermBlockTimeBuffer(t *testing.T) {
	oneYearHeight := uint64(365 * 24 * 60 * 60 / 5)
	oneYearSeconds := time.Duration(oneYearHeight) * 5 * time.Second

	oneYearHeight++
	oneYearSeconds += 6 * time.Second

	require.Equal(t, 4*time.Second, calculateBlockTimeBuffer(
		oneYearHeight,
		5*time.Second,
		oneYearSeconds,
		0*time.Second,
		1*time.Second,
	))
}
