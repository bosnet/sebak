package sebak

import (
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyCreateAccount struct {
	Target string             `json:"target"`
	Amount sebakcommon.Amount `json:"amount"`
}

func NewOperationBodyCreateAccount(target string, amount sebakcommon.Amount) OperationBodyCreateAccount {
	return OperationBodyCreateAccount{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyCreateAccount) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`: lower than 1")
		return
	}

	return
}

func (o OperationBodyCreateAccount) Validate(st sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Target` is not in `Block Account`

	return
}

func (o OperationBodyCreateAccount) TargetAddress() string {
	return o.Target
}

func (o OperationBodyCreateAccount) GetAmount() sebakcommon.Amount {
	return o.Amount
}

func FinishOperationCreateAccountWithStateDB(sdb *statedb.StateDB, tx Transaction, op Operation) (err error) {
	var sourceAddr, targetAddr string
	sourceAddr = tx.B.Source
	targetAddr = op.B.TargetAddress()

	if sdb.ExistAccount(sourceAddr) == false {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if sdb.ExistAccount(targetAddr) == true {
		err = sebakerror.ErrorBlockAccountAlreadyExists
		return
	}
	sdb.AddBalanceWithCheckpoint(targetAddr, op.B.GetAmount(), tx.NextTargetCheckpoint())

	log.Debug("new account created", "source", sourceAddr, "target", targetAddr)

	return
}
