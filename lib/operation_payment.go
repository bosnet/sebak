package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyPayment struct {
	Target string             `json:"target"`
	Amount sebakcommon.Amount `json:"amount"`
}

func NewOperationBodyPayment(target string, amount sebakcommon.Amount) OperationBodyPayment {
	return OperationBodyPayment{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyPayment) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyPayment) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	return
}

func (o OperationBodyPayment) Validate(st sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Target` is in `Block Account`
	// TODO check over minimum balance
	return
}

func (o OperationBodyPayment) TargetAddress() string {
	return o.Target
}

func (o OperationBodyPayment) GetAmount() sebakcommon.Amount {
	return o.Amount
}

func FinishOperationPaymentWithStateDB(sdb *statedb.StateDB, tx Transaction, op Operation) (err error) {
	var sourceAddr, targetAddr string
	sourceAddr = tx.B.Source
	targetAddr = op.B.TargetAddress()

	if sdb.ExistAccount(sourceAddr) == false {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if sdb.ExistAccount(targetAddr) == false {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}

	current, err := sebakcommon.ParseCheckpoint(sdb.GetCheckPoint(targetAddr))
	next, err := sebakcommon.ParseCheckpoint(tx.NextTargetCheckpoint())
	newCheckPoint := sebakcommon.MakeCheckpoint(current[0], next[1])

	sdb.AddBalanceWithCheckpoint(targetAddr, op.B.GetAmount(), newCheckPoint)

	log.Debug("payment done", "source", sourceAddr, "target", targetAddr, "amount", op.B.GetAmount())

	return
}
