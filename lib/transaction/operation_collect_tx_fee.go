package transaction

import (
	"encoding/json"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

// OperationBodyTransactionFee is the operation to send the collected transacton
// fee to certain account. To prevent the hash duplication of transaction,
// OperationBodyTransactionFee has block related data.
type OperationBodyCollectTxFee struct {
	Target      string        `json:"target"`
	Amount      common.Amount `json:"amount"`
	Txs         uint64        `json:"txs"`
	BlockHeight uint64        `json:"block-height"`
	BlockHash   string        `json:"block-hash"`
	TotalTxs    uint64        `json:"total-txs"`
}

func NewOperationBodyCollectTxFee(
	target string,
	amount common.Amount,
	txs uint64,
	blockHeight uint64,
	blockHash string,
	totalTxs uint64,
) OperationBodyCollectTxFee {
	return OperationBodyCollectTxFee{
		Target:      target,
		Amount:      amount,
		Txs:         txs,
		BlockHeight: blockHeight,
		BlockHash:   blockHash,
		TotalTxs:    totalTxs,
	}
}

func (o OperationBodyCollectTxFee) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if len(o.BlockHash) < 1 {
		err = errors.ErrorInvalidOperation
		return
	}

	if int64(o.Txs) > 0 && int64(o.Amount) < 1 {
		err = errors.ErrorOperationAmountUnderflow
		return
	}

	if int64(o.Txs) == 0 && int64(o.Amount) != 0 {
		err = errors.ErrorOperationAmountOverflow
		return
	}

	if o.Txs < 1 {
		if o.Amount != 0 {
			err = errors.ErrorInvalidOperation
			return
		}
	} else if o.Amount < (common.BaseFee * common.Amount(o.Txs)) {
		err = errors.ErrorInvalidOperation
		return
	}

	return
}

func (o OperationBodyCollectTxFee) TargetAddress() string {
	return o.Target
}

func (o OperationBodyCollectTxFee) GetAmount() common.Amount {
	return o.Amount
}

func (o OperationBodyCollectTxFee) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
