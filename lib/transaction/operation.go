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
	OperationCongressVoting                     = "congress-voting"
	OperationCongressVotingResult               = "congress-voting-result"
	OperationCollectTxFee                       = "collect-tx-fee"
	OperationMembershipRegister                 = "register membership"
	OperationMembershipDeregister               = "deregister membership"
)

var OperationTypesNormalTransaction map[OperationType]struct{} = map[OperationType]struct{}{
	OperationCreateAccount:        struct{}{},
	OperationPayment:              struct{}{},
	OperationCongressVoting:       struct{}{},
	OperationCongressVotingResult: struct{}{},
	OperationMembershipRegister:   struct{}{},
	OperationMembershipDeregister: struct{}{},
}

type Operation struct {
	H OperationHeader
	B OperationBody
}

func NewOperation(opb OperationBody) (op Operation, err error) {
	var t OperationType
	switch opb.(type) {
	case OperationBodyCreateAccount:
		t = OperationCreateAccount
	case OperationBodyPayment:
		t = OperationPayment
	case OperationBodyCollectTxFee:
		t = OperationCollectTxFee
	default:
		err = errors.ErrorUnknownOperationType
		return
	}

	op = Operation{
		H: OperationHeader{Type: t},
		B: opb,
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

type operationEnvelop struct {
	H OperationHeader
	B interface{}
}

func (o *Operation) UnmarshalJSON(b []byte) (err error) {
	var envelop json.RawMessage
	oj := operationEnvelop{
		B: &envelop,
	}
	if err = json.Unmarshal(b, &oj); err != nil {
		return
	}

	o.H = oj.H

	var body OperationBody
	if body, err = UnmarshalOperationBodyJSON(oj.H.Type, envelop); err != nil {
		return
	}
	o.B = body

	return
}

func UnmarshalOperationBodyJSON(t OperationType, b []byte) (body OperationBody, err error) {
	switch t {
	case OperationCreateAccount:
		var ob OperationBodyCreateAccount
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationPayment:
		var ob OperationBodyPayment
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationCongressVoting:
		var ob OperationBodyCongressVoting
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationCongressVotingResult:
		var ob OperationBodyCongressVotingResult
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationCollectTxFee:
		var ob OperationBodyCollectTxFee
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationMembershipRegister:
		var ob OperationBodyMembershipRegister
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case OperationMembershipDeregister:
		var ob OperationBodyMembershipDeregister
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	default:
		err = errors.ErrorInvalidOperation
		return
	}

	return
}
