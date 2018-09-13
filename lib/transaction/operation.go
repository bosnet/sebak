package transaction

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
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
	return common.MustMakeObjectHash(o)
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

func NewOperationFromInterface(oj OperationFromJSON) (Operation, error) {
	var op Operation
	op.H = oj.H

	body := oj.B.(map[string]interface{})
	switch op.H.Type {
	case OperationCreateAccount:
		if amount, err := common.AmountFromString(fmt.Sprintf("%v", body["amount"])); err != nil {
			return Operation{}, err
		} else if target, is_target := body["target"].(string); !is_target {
			return Operation{}, errors.ErrorInvalidOperation
		} else {
			op.B = NewOperationBodyCreateAccount(target, amount)
		}
	case OperationPayment:
		if amount, err := common.AmountFromString(fmt.Sprintf("%v", body["amount"])); err != nil {
			return Operation{}, err
		} else if target, is_target := body["target"].(string); !is_target {
			return Operation{}, errors.ErrorInvalidOperation
		} else {
			op.B = NewOperationBodyPayment(target, amount)
		}
	default:
		return Operation{}, errors.ErrorInvalidOperation
	}

	return op, nil
}

func NewOperation(t OperationType, body OperationBody) (op Operation, err error) {
	if err = body.IsWellFormed([]byte("")); err != nil {
		return
	}

	switch t {
	case OperationCreateAccount:
		if _, ok := body.(OperationBodyCreateAccount); !ok {
			err = errors.ErrorTypeOperationBodyNotMatched
			return
		}
	case OperationPayment:
		if _, ok := body.(OperationBodyPayment); !ok {
			err = errors.ErrorTypeOperationBodyNotMatched
			return
		}
	default:
		err = errors.ErrorUnknownOperationType
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
	//
	// Check that this transaction is self consistent
	//
	// This routine is used by the transaction checker and thus is part of consensus
	//
	// Params:
	//   networkid = Network id this operation was emitted on
	//
	// Returns:
	//   An `error` if that transaction is invalid, `nil` otherwise
	//
	IsWellFormed(networkid []byte) error
	TargetAddress() string
	GetAmount() common.Amount
}
