package consensus

import (
	"sort"

	"boscoin.io/sebak/lib/network"
)

type ProposerSelector interface {
	Select(uint64, uint64) string
}

type SequentialSelector struct {
	cm network.ConnectionManager
}

func NewSequentialSelector(cm network.ConnectionManager) SequentialSelector {
	p := SequentialSelector{cm}
	return p
}

func (s SequentialSelector) Select(blockHeight uint64, roundNumber uint64) string {
	candidates := sort.StringSlice(s.cm.AllValidators())
	candidates.Sort()
	return candidates[(blockHeight+roundNumber)%uint64(len(candidates))]
}
