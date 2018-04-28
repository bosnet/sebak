package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
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

func (o Operation) MakeHash() []byte {
	return util.MustMakeObjectHash(o)
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

func (o Operation) Validate(st storage.LevelDBBackend) (err error) {
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

type OperationHeader struct {
	Type OperationType `json:"type"`
}

type OperationBody interface {
	Validate() error
	IsWellFormed() error
	GetTargetAddress() string
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

func (ob OperationBodyPayment) IsWellFormed() (err error) {
	if _, err = keypair.Parse(ob.Target); err != nil {
		return
	}

	if int64(ob.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	return
}

func (ob OperationBodyPayment) Validate() (err error) {
	// TODO check whether `Target` is in `Block Account`

	return
}

func (ob OperationBodyPayment) GetTargetAddress() string {
	return ob.Target
}

func (ob OperationBodyPayment) GetAmount() Amount {
	return ob.Amount
}

type OperationBodyCreateAccount struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

func (ob OperationBodyCreateAccount) IsWellFormed() (err error) {
	if _, err = keypair.Parse(ob.Target); err != nil {
		return
	}

	if int64(ob.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`: lower than 1")
		return
	} // TODO check over minimum balance

	return
}

func (ob OperationBodyCreateAccount) Validate() (err error) {
	// TODO check whether `Target` is not in `Block Account`

	return
}

func (ob OperationBodyCreateAccount) GetTargetAddress() string {
	return ob.Target
}

func (ob OperationBodyCreateAccount) GetAmount() Amount {
	return ob.Amount
}
