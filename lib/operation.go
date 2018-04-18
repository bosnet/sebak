package sebak

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
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

func (o Operation) IsWellFormed() (err error) {
	if len(o.H.Hash) < 1 {
		err = fmt.Errorf("empty `H.Hash`")
		return
	}

	if err = o.B.IsWellFormed(); err != nil {
		return
	}

	return
}

func (o Operation) Validate() (err error) {
	if o.B.GetHashString() != o.H.Hash {
		return fmt.Errorf("`Hash` mismatch")
	}

	if err = o.B.Validate(); err != nil {
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
		//
	case OperationPayment:
		op.B = OperationBodyPayment{
			Receiver: body["receiver"].(string),
			Amount:   body["amount"].(string),
		}
		if err != nil {
			return
		}
	}

	return
}

type OperationHeader struct {
	Hash string        `json:"hash"`
	Type OperationType `json:"type"`
}

type OperationBody interface {
	GetHash() []byte
	GetHashString() string
	Validate() error
	IsWellFormed() error
	GetTargetAddress() string
	GetAmount() int64
}

type OperationBodyPayment struct {
	Receiver string `json:"receiver"`
	Amount   string `json:"amount"`
}

func (ob OperationBodyPayment) IsWellFormed() (err error) {
	if _, err = keypair.Parse(ob.Receiver); err != nil {
		return
	}

	var i int64
	if i, err = strconv.ParseInt(ob.Amount, 10, 64); err != nil {
		return
	} else if i < 1 {
		err = fmt.Errorf("invalid `Amount`: %d < 1", i)
		return
	}

	return
}

func (ob OperationBodyPayment) GetHash() []byte {
	encoded, _ := GetObjectHash(ob)
	return encoded
}

func (ob OperationBodyPayment) GetHashString() string {
	return base58.Encode(ob.GetHash())
}

func (ob OperationBodyPayment) Validate() (err error) {
	// TODO check whether `Receiver` is in `Block Account`

	return
}

func (ob OperationBodyPayment) GetTargetAddress() string {
	return ob.Receiver
}

func (ob OperationBodyPayment) GetAmount() (a int64) {
	a, _ = strconv.ParseInt(ob.Amount, 10, 64)
	return
}
