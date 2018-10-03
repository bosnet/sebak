package transaction

import (
	"encoding/json"
	"strings"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

// OperationBodyInflation is the operation to raise inflation in every block. To
// prevent the hash duplication of transaction, OperationBodyInflation has block
// related data.
type OperationBodyInflation struct {
	Target         string        `json:"target"`
	Amount         common.Amount `json:"amount"`
	InitialBalance common.Amount `json:"initial-balance"`
	Ratio          string        `json:"ratio"`
	BlockHeight    uint64        `json:"block-height"`
	BlockHash      string        `json:"block-hash"`
	TotalTxs       uint64        `json:"total-txs"`
}

func NewOperationBodyInflation(
	target string,
	amount common.Amount,
	initialBalance common.Amount,
	ratio float64,
	blockHeight uint64,
	blockHash string,
	totalTxs uint64,
) OperationBodyInflation {
	return OperationBodyInflation{
		Target:         target,
		Amount:         amount,
		InitialBalance: initialBalance,
		Ratio:          common.InflationRatio2String(ratio),
		BlockHeight:    blockHeight,
		BlockHash:      blockHash,
		TotalTxs:       totalTxs,
	}
}

func (o OperationBodyInflation) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if len(o.BlockHash) < 1 {
		err = errors.ErrorInvalidOperation
		return
	}

	if o.InitialBalance < 0 {
		err = errors.ErrorInvalidOperation
		return
	}

	var ratio float64
	if !strings.HasPrefix(o.Ratio, "0.") {
		err = errors.ErrorInvalidOperation
		return
	} else if ratio, err = common.String2InflationRatio(o.Ratio); err != nil {
		return
	} else if ratio < 0 || ratio > 1 {
		err = errors.ErrorInvalidOperation
		return
	}

	return
}

func (o OperationBodyInflation) TargetAddress() string {
	return o.Target
}

func (o OperationBodyInflation) GetAmount() common.Amount {
	return o.Amount
}

func (o OperationBodyInflation) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
