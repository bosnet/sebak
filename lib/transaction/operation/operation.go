package operation

import (
	"encoding/json"
	"io"
	"reflect"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

type OperationType byte

const (
	TypeCreateAccount OperationType = iota
	TypePayment
	TypeCongressVoting
	TypeCongressVotingResult
	TypeCollectTxFee
	TypeInflation
	TypeUnfreezingRequest
	TypeInflationPF
)

var (
	typeName []string = []string{
		"create-account",
		"payment",
		"congress-voting",
		"congress-voting-result",
		"collect-tx-fee",
		"inflation",
		"unfreezing-request",
		"inflation-pf",
	}
)

// Implement `fmt.Stringer`
func (ot OperationType) String() string {
	return typeName[ot]
}

// Implement encoding.TextMarshaler
// Also used in the API to avoid breaking clients
func (ot OperationType) MarshalText() (text []byte, err error) {
	return []byte(ot.String()), nil
}

// Implement encoding.TextUnmarshaler
func (ot *OperationType) UnmarshalText(text []byte) (err error) {
	if idx, found := common.InStringArray(typeName, string(text)); !found {
		return errors.InvalidOperation
	} else {
		*ot = OperationType(idx)
		return nil
	}
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

// Implement `common.Encoder`
// The implementation MUST use value type, not pointer
func (op Operation) EncodeRLP(w io.Writer) error {
	typeList := struct{ Value string }{Value: op.H.Type.String()}
	if s1, r1, e1 := common.EncodeToReader(typeList); e1 != nil {
		return e1
	} else if s2, r2, e2 := common.EncodeToReader(op.B); e2 != nil {
		return e2
	} else {
		// Write it as a list
		totalLength := uint64(s1) + uint64(s2)
		if err := common.PutListLength(w, totalLength); err != nil {
			return err
		}
		if _, err := io.Copy(w, r1); err != nil {
			return err
		}
		if _, err := io.Copy(w, r2); err != nil {
			return err
		}
		return nil
	}
}

// Implement `common.Decoder`
func (op *Operation) DecodeRLP(s *common.RLPStream) error {
	// The value is encoded as a list
	if _, err := s.List(); err != nil {
		return err
	}

	// Read operation type [ "..." ]
	if _, err := s.List(); err != nil {
		return err
	} else if typeBytes, err := s.Bytes(); err != nil {
		return err
	} else if err = op.H.Type.UnmarshalText(typeBytes); err != nil {
		return err
	} else if err = s.ListEnd(); err != nil {
		return err
	}

	// Read operation body
	// the struct is encoded as a list and handled by `Decode`
	if ob, err := newBodyFromType(op.H.Type); err != nil {
		return err
	} else if err = s.Decode(ob); err != nil {
		return err
	} else {
		op.B = reflect.ValueOf(ob).Elem().Interface().(Body)
	}

	// Close the list
	if err := s.ListEnd(); err != nil {
		return err
	}

	return nil
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
