package voting

import "strconv"

type Basis struct {
	Round     uint64 `json:"round"`      // round sequence number
	Height    uint64 `json:"height"`     // last block height
	BlockHash string `json:"block-hash"` // hash of last block
	TotalTxs  uint64 `json:"total-txs"`
	TotalOps  uint64 `json:"total-ops"`
}

func (r Basis) Index() string {
	return strconv.FormatUint(r.Height, 10) + "-" + strconv.FormatUint(r.Round, 10)
}
