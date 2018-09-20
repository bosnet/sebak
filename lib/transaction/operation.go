package transaction

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type OperationType string

const (
	OperationCreateAccount        OperationType = "create-account"
	OperationPayment                            = "payment"
	OperationCongressVoting                     = "cv"
	OperationCongressVotingResult               = "cvr"
)

type Operation struct {
	H OperationHeader
	B OperationBody
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
	IsWellFormed([]byte) error
	Serialize() ([]byte, error)
}

type OperationBodyPayable interface {
	OperationBody
	TargetAddress() string
	GetAmount() common.Amount
}

func (o Operation) MakeHash() []byte {
	return common.MustMakeObjectHash(o)
}

func (o Operation) MakeHashString() string {
	return base58.Encode(o.MakeHash())
}

func (o Operation) IsWellFormed(networkID []byte) (err error) {
	return o.B.IsWellFormed(networkID)
}

func (o Operation) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

func (o Operation) String() string {
	encoded, _ := json.MarshalIndent(o, "", "  ")

	return string(encoded)
}

type OperationEnvelop struct {
	H OperationHeader
	B interface{}
}

func (o *Operation) UnmarshalJSON(b []byte) (err error) {
	var envelop json.RawMessage
	oj := OperationEnvelop{
		B: &envelop,
	}
	if err = json.Unmarshal(b, &oj); err != nil {
		return
	}

	o.H = oj.H

	switch oj.H.Type {
	case OperationCreateAccount:
		var body OperationBodyCreateAccount
		if err = json.Unmarshal(envelop, &body); err != nil {
			return
		}
		o.B = body
	case OperationPayment:
		var body OperationBodyPayment
		if err = json.Unmarshal(envelop, &body); err != nil {
			return
		}
		o.B = body
	case OperationCongressVoting:
		var body OperationBodyCongressVoting
		if err = json.Unmarshal(envelop, &body); err != nil {
			return
		}
		o.B = body
	case OperationCongressVotingResult:
		var body OperationBodyCongressVotingResult
		if err = json.Unmarshal(envelop, &body); err != nil {
			return
		}
		o.B = body
	default:
		return errors.ErrorInvalidOperation
	}
	return
}
