package sebak

import (
	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
)

type Round struct {
	Number      uint64 `json:"number"`       // round sequence number
	BlockHeight uint64 `json:"block-height"` // last block height
	BlockHash   string `json:"block-hash"`   // hash of last block
	TotalTxs    uint64 `json:"total-txs"`
}

func (r Round) Hash() string {
	return base58.Encode(sebakcommon.MustMakeObjectHash(r))
}

func (r Round) IsSame(a Round) bool {
	return r.Hash() == a.Hash()
}
