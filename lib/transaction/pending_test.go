package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPendingPool(t *testing.T) {
	pp := NewPendingPool()

	// Insert in order
	pp.Insert(42, "op1")
	pp.Insert(50, "op2")
	pp.Insert(100, "op3")

	// Then out of order
	pp.Insert(200, "op4")
	pp.Insert(100, "op5")
	pp.Insert(1, "op6")

	resultIdx := 0
	results := [6]pendingPoolItem{
		{1, "op6"},
		{42, "op1"},
		{50, "op2"},
		{100, "op3"},
		{100, "op5"},
		{200, "op4"},
	}

	offset := uint64(0)
	for i := uint64(0); i < 250; i++ {
		key := pp.Peek(i, offset)
		if resultIdx >= len(results) {
			// If we're past the last expected result (height=200)
			require.Equal(t, len(key), 0)
			pp.PopHeight(i)
			offset = 0
		} else if results[resultIdx].height == i {
			require.Equal(t, results[resultIdx].opKey, key)
			resultIdx += 1
			i -= 1 // Make sure we pop all entries at the same height
			offset += 1
		} else {
			require.Equal(t, len(key), 0)
			pp.PopHeight(i)
			offset = 0
		}
	}
}
