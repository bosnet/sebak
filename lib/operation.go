package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
)

type OperationType string

const (
	OperationCreateAccount OperationType = "create-account"
	OperationPayment                     = "payment"
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

func (o Operation) IsWellFormed() (err error) {
	if err = o.B.IsWellFormed(); err != nil {
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
		var amount Amount
		amount, err = AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = NewOperationBodyCreateAccount(body["target"].(string), amount)
	case OperationPayment:
		var amount Amount
		amount, err = AmountFromString(fmt.Sprintf("%v", body["amount"]))
		if err != nil {
			return
		}
		op.B = OperationBodyPayment{
			Target: body["target"].(string),
			Amount: amount,
		}
		if err != nil {
			return
		}
	}

	return
}

func NewOperation(t OperationType, body OperationBody) (op Operation, err error) {
	if err = body.IsWellFormed(); err != nil {
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
	IsWellFormed() error
	TargetAddress() string
	GetAmount() Amount
}

type OperationBodyPayment struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

func (o OperationBodyPayment) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyPayment) IsWellFormed() (err error) {
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

func (o OperationBodyPayment) GetAmount() Amount {
	return o.Amount
}

type OperationBodyCreateAccount struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

func NewOperationBodyCreateAccount(target string, amount Amount) OperationBodyCreateAccount {
	return OperationBodyCreateAccount{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyCreateAccount) IsWellFormed() (err error) {
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

func (o OperationBodyCreateAccount) GetAmount() Amount {
	return o.Amount
}
