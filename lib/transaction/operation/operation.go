package operation

import (
	"encoding/json"
	"reflect"

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
	TypeInflationPF          OperationType = "inflation-pf"
)

func IsValidOperationType(oType string) bool {
	_, b := common.InStringArray([]string{
		string(TypeCreateAccount),
		string(TypePayment),
		string(TypeCongressVoting),
		string(TypeCongressVotingResult),
		string(TypeCollectTxFee),
		string(TypeInflation),
		string(TypeUnfreezingRequest),
		string(TypeInflationPF),
	}, oType)
	return b
}

func IsNormalOperation(t OperationType) bool {
	switch t {
	case TypeCreateAccount, TypePayment,
		TypeCongressVoting, TypeCongressVotingResult,
		TypeUnfreezingRequest, TypeInflationPF:
		return true
	default:
		return false
	}
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
	case InflationPF:
		t = TypeInflationPF
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
	//   config = Consensus configuration
	//
	// Returns:
	//   An `error` if that transaction is invalid, `nil` otherwise
	//
	IsWellFormed(common.Config) error
	HasFee() bool
}

type Payable interface {
	Body
	TargetAddress() string
	GetAmount() common.Amount
}

type Targetable interface {
	TargetAddress() string
}

func (o Operation) IsWellFormed(conf common.Config) (err error) {
	return o.B.IsWellFormed(conf)
}

func (o Operation) String() string {
	encoded, _ := json.MarshalIndent(o, "", "  ")

	return string(encoded)
}

func (o Operation) HasFee() bool {
	return o.B.HasFee()
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
	return nil
}

func UnmarshalBodyJSON(t OperationType, b []byte) (Body, error) {
	if bi, err := newBodyFromType(t); err != nil {
		return nil, err
	} else if err = json.Unmarshal(b, bi); err != nil {
		return nil, err
	} else {
		// No other way to go from interface-to-pointer to interface-to-value
		// because values within interfaces are not addressable
		return reflect.ValueOf(bi).Elem().Interface().(Body), nil
	}
}

// Returns: A pointer to a body with a type matching `ty`
func newBodyFromType(ty OperationType) (interface{}, error) {
	switch ty {
	case TypeCreateAccount:
		return &CreateAccount{}, nil
	case TypePayment:
		return &Payment{}, nil
	case TypeCongressVoting:
		return &CongressVoting{}, nil
	case TypeCongressVotingResult:
		return &CongressVotingResult{}, nil
	case TypeCollectTxFee:
		return &CollectTxFee{}, nil
	case TypeInflation:
		return &Inflation{}, nil
	case TypeUnfreezingRequest:
		return &UnfreezeRequest{}, nil
	case TypeInflationPF:
		return &InflationPF{}, nil
	default:
		return nil, errors.InvalidOperation
	}
}
