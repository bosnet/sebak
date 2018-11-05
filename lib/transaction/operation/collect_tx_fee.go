package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

// OperationBodyTransactionFee is the operation to send the collected transacton
// fee to certain account. To prevent the hash duplication of transaction,
// OperationBodyTransactionFee has block related data.
type CollectTxFee struct {
	Target    string        `json:"target"`
	Amount    common.Amount `json:"amount"`
	Txs       uint64        `json:"txs"`
	Height    uint64        `json:"block-height"`
	BlockHash string        `json:"block-hash"`
	TotalTxs  uint64        `json:"total-txs"`
	TotalOps  uint64        `json:"total-ops"`
}

func NewCollectTxFee(
	target string,
	amount common.Amount,
	txs uint64,
	blockHeight uint64,
	blockHash string,
	totalTxs uint64,
) CollectTxFee {
	return CollectTxFee{
		Target:    target,
		Amount:    amount,
		Txs:       txs,
		Height:    blockHeight,
		BlockHash: blockHash,
		TotalTxs:  totalTxs,
	}
}

func (o CollectTxFee) IsWellFormed(common.Config) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if len(o.BlockHash) < 1 {
		err = errors.InvalidOperation
		return
	}

	if int64(o.Txs) > 0 && int64(o.Amount) < 1 {
		err = errors.OperationAmountUnderflow
		return
	}

	if int64(o.Txs) == 0 && int64(o.Amount) != 0 {
		err = errors.OperationAmountOverflow
		return
	}

	if o.Txs < 1 {
		if o.Amount != 0 {
			err = errors.InvalidOperation
			return
		}
	} else if o.Amount < (common.BaseFee * common.Amount(o.Txs)) {
		err = errors.InvalidOperation
		return
	}

	return
}

func (o CollectTxFee) TargetAddress() string {
	return o.Target
}

func (o CollectTxFee) GetAmount() common.Amount {
	return o.Amount
}

func (o CollectTxFee) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
