package operation

import (
	"encoding/json"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

// Inflation is the operation to raise inflation in every block. To
// prevent the hash duplication of transaction, Inflation has block
// related data.
type Inflation struct {
	Target         string        `json:"target"`
	Amount         common.Amount `json:"amount"`
	InitialBalance common.Amount `json:"initial_balance"`
	Ratio          string        `json:"ratio"`
	Height         uint64        `json:"block-height"`
	BlockHash      string        `json:"block-hash"`
	TotalTxs       uint64        `json:"total-txs"`
	TotalOps       uint64        `json:"total-ops"`
}

func NewOperationBodyInflation(
	target string,
	amount common.Amount,
	initialBalance common.Amount,
	blockHeight uint64,
	blockHash string,
	totalTxs uint64,
) Inflation {
	return Inflation{
		Target:         target,
		Amount:         amount,
		InitialBalance: initialBalance,
		Ratio:          common.InflationRatioString,
		Height:         blockHeight,
		BlockHash:      blockHash,
		TotalTxs:       totalTxs,
	}
}

func (o Inflation) IsWellFormed(common.Config) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if len(o.BlockHash) < 1 {
		err = errors.InvalidOperation
		return
	}

	if o.InitialBalance < 0 {
		err = errors.InvalidOperation
		return
	}

	var ratio float64
	if !strings.HasPrefix(o.Ratio, "0.") {
		err = errors.InvalidOperation
		return
	} else if ratio, err = common.String2InflationRatio(o.Ratio); err != nil {
		return
	} else if ratio < 0 || ratio > 1 {
		err = errors.InvalidOperation
		return
	}

	return
}

func (o Inflation) TargetAddress() string {
	return o.Target
}

func (o Inflation) GetAmount() common.Amount {
	return o.Amount
}

func (o Inflation) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
