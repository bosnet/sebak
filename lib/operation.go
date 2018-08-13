package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type OperationType string

const (
	OperationCreateAccount   OperationType = "create-account"
	OperationPayment                       = "payment"
	OperationContractDeploy                = "contract-deploy"
	OperationContractExecute               = "contract-execute"
)

type Operation struct {
	H OperationHeader
	B OperationBody
}

func (o Operation) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(o)
}

func (o Operation) MakeHashString() string {
	return base58.Encode(o.MakeHash())
}

func (o Operation) IsWellFormed(networkID []byte) (err error) {
	if err = o.B.IsWellFormed(networkID); err != nil {
		return
	}

	return
}

func (o Operation) Validate(st sebakstorage.LevelDBBackend) (err error) {
	if err = o.B.Validate(st); err != nil {
		return
	}

	return
}

func (o Operation) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o Operation) String() string {
	encoded, _ := json.MarshalIndent(o, "", "  ")

	return string(encoded)
}

type OperationFromJSON struct {
	H OperationHeader
	B interface{}
}

func NewOperationFromBytes(b []byte) (op Operation, err error) {
	var oj OperationFromJSON

	if err = json.Unmarshal(b, &oj); err != nil {
		return
	}

	op, err = NewOperationFromInterface(oj)

	return
}

func NewOperationFromInterface(oj OperationFromJSON) (op Operation, err error) {
	op.H = oj.H

	body := oj.B.(map[string]interface{})
	switch op.H.Type {
	case OperationCreateAccount:
		var amount sebakcommon.Amount
		amount, err = sebakcommon.AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = NewOperationBodyCreateAccount(body["target"].(string), amount)
	case OperationPayment:
		var amount sebakcommon.Amount
		amount, err = sebakcommon.AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = NewOperationBodyPayment(body["target"].(string), amount)
	case OperationContractExecute:
		var amount sebakcommon.Amount
		amount, err = sebakcommon.AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = NewOperationBodyContractExecute(body["target"].(string), amount, body["method"].(string), body["args"].([]string))
	case OperationContractDeploy:
		var amount sebakcommon.Amount
		amount, err = sebakcommon.AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = NewOperationBodyContractDeploy(body["target"].(string), amount, body["codeType"].(int), body["code"].(string))
	}

	return
}

func NewOperation(t OperationType, body OperationBody) (op Operation, err error) {
	if err = body.IsWellFormed([]byte("")); err != nil {
		return
	}

	switch t {
	case OperationCreateAccount:
		if _, ok := body.(OperationBodyCreateAccount); !ok {
			err = sebakerror.ErrorTypeOperationBodyNotMatched
			return
		}
	case OperationPayment:
		if _, ok := body.(OperationBodyPayment); !ok {
			err = sebakerror.ErrorTypeOperationBodyNotMatched
			return
		}
	default:
		err = sebakerror.ErrorUnknownOperationType
		return
	}

	op = Operation{
		H: OperationHeader{Type: t},
		B: body,
	}
	return
}

type OperationHeader struct {
	Type OperationType `json:"type"`
}

type OperationBody interface {
	Validate(sebakstorage.LevelDBBackend) error
	IsWellFormed([]byte) error
	TargetAddress() string
	GetAmount() sebakcommon.Amount
}

// FinishOperation do finish the task after consensus by the type of each operation.
func FinishOperation(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	switch op.H.Type {
	case OperationCreateAccount:
		return FinishOperationCreateAccount(st, tx, op)
	case OperationPayment:
		return FinishOperationPayment(st, tx, op)
	case OperationContractExecute:
		return FinishOperationBodyContractDeploy(st, tx, op)
	default:
		err = sebakerror.ErrorUnknownOperationType
		return
	}

	return
}
