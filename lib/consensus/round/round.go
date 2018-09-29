package round

import "strconv"

type Round struct {
	Number      uint64 `json:"number"`       // round sequence number
	BlockHeight uint64 `json:"block-height"` // last block height
	BlockHash   string `json:"block-hash"`   // hash of last block
	TotalTxs    uint64 `json:"total-txs"`
}

func (r Round) Hash() string {
	// return base58.Encode(common.MustMakeObjectHash(r))
	return strconv.FormatUint(r.BlockHeight, 10)
}
