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

func (s SequentialSelector) Select(blockHeight uint64, round uint64) string {
	candidates := sort.StringSlice(s.cm.AllValidators())
	candidates.Sort()
	return candidates[(blockHeight+round)%uint64(len(candidates))]
}
