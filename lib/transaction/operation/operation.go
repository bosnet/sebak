package operation

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

type OperationType string

const (
	TypeCreateAccount        OperationType = "create-account"
	TypePayment              OperationType = "payment"
	TypeCongressVoting       OperationType = "congress-voting"
	TypeCongressVotingResult OperationType = "congress-voting-result"
	TypeCollectTxFee         OperationType = "collect-tx-fee"
	TypeInflation            OperationType = "inflation"
	TypeUnfreezingRequest    OperationType = "unfreezing-request"
)

func IsValidOperationType(oType string) bool {
	_, b := common.InStringArray([]string{
		string(TypeCreateAccount),
		string(TypePayment),
		string(TypeCongressVoting),
		string(TypeCongressVotingResult),
		string(TypeCollectTxFee),
		string(TypeInflation),
	}, oType)
	return b
}

var KindsNormalTransaction map[OperationType]struct{} = map[OperationType]struct{}{
	TypeCreateAccount:        struct{}{},
	TypePayment:              struct{}{},
	TypeCongressVoting:       struct{}{},
	TypeCongressVotingResult: struct{}{},
	TypeUnfreezingRequest:    struct{}{},
}

type Operation struct {
	H Header
	B Body
}

func NewOperation(opb Body) (op Operation, err error) {
	var t OperationType
	switch opb.(type) {
	case CreateAccount:
		t = TypeCreateAccount
	case Payment:
		t = TypePayment
	case CollectTxFee:
		t = TypeCollectTxFee
	case Inflation:
		t = TypeInflation
	case UnfreezeRequest:
		t = TypeUnfreezingRequest
	case CongressVoting:
		t = TypeCongressVoting
	case CongressVotingResult:
		t = TypeCongressVotingResult
	default:
		err = errors.UnknownOperationType
		return
	}

	op = Operation{
		H: Header{Type: t},
		B: opb,
	}

	return
}

type Header struct {
	Type OperationType `json:"type"`
}

type Body interface {
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
	IsWellFormed([]byte, common.Config) error
	Serialize() ([]byte, error)
}

type Payable interface {
	Body
	TargetAddress() string
	GetAmount() common.Amount
}

func (o Operation) MakeHash() []byte {
	return common.MustMakeObjectHash(o)
}

func (o Operation) MakeHashString() string {
	return base58.Encode(o.MakeHash())
}

func (o Operation) IsWellFormed(networkID []byte, conf common.Config) (err error) {
	return o.B.IsWellFormed(networkID, conf)
}

func (o Operation) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

func (o Operation) String() string {
	encoded, _ := json.MarshalIndent(o, "", "  ")

	return string(encoded)
}

type envelop struct {
	H Header
	B interface{}
}

func (o *Operation) UnmarshalJSON(b []byte) (err error) {
	var raw json.RawMessage
	oj := envelop{
		B: &raw,
	}
	if err = json.Unmarshal(b, &oj); err != nil {
		return
	}

	o.H = oj.H

	var body Body
	if body, err = UnmarshalBodyJSON(oj.H.Type, raw); err != nil {
		return
	}
	o.B = body

	return
}

func UnmarshalBodyJSON(t OperationType, b []byte) (body Body, err error) {
	switch t {
	case TypeCreateAccount:
		var ob CreateAccount
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypePayment:
		var ob Payment
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypeCongressVoting:
		var ob CongressVoting
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypeCongressVotingResult:
		var ob CongressVotingResult
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypeCollectTxFee:
		var ob CollectTxFee
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypeInflation:
		var ob Inflation
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	case TypeUnfreezingRequest:
		var ob UnfreezeRequest
		if err = json.Unmarshal(b, &ob); err != nil {
			return
		}
		body = ob
	default:
		err = errors.InvalidOperation
		return
	}

	return
}
