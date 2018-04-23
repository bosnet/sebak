package sebak

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

type Transaction struct {
	H TransactionHeader
	B TransactionBody
}

type TransactionFromJSON struct {
	H TransactionHeader
	B TransactionBodyFromJSON
}

type TransactionBodyFromJSON struct {
	Source     string              `json:"source"`
	Fee        Amount              `json:"fee"`
	Checkpoint string              `json:"checkpoint"`
	Operations []OperationFromJSON `json:"operations"`
}

func NewTransactionFromJSON(b []byte) (tx Transaction, err error) {
	var txt TransactionFromJSON
	if err = json.Unmarshal(b, &txt); err != nil {
		return
	}

	var operations []Operation
	for _, o := range txt.B.Operations {
		var op Operation
		if op, err = NewOperationFromInterface(o); err != nil {
			return
		}
		operations = append(operations, op)
	}

	tx.H = txt.H
	tx.B = TransactionBody{
		Source:     txt.B.Source,
		Fee:        txt.B.Fee,
		Checkpoint: txt.B.Checkpoint,
		Operations: operations,
	}

	return
}

func (o Transaction) IsWellFormed() (err error) {
	// TODO check `Version` format with SemVer

	if _, err = keypair.Parse(o.B.Source); err != nil {
		err = fmt.Errorf("invalid `Source`: %v", err)
		return
	}

	if int64(o.B.Fee) < BaseFee {
		err = errors.New("`fee` must be greater than `BaseFee`")
		return
	}

	for _, op := range o.B.Operations {
		if ta := op.B.GetTargetAddress(); o.B.Source == ta {
			err = fmt.Errorf("`Operation.TargetAddress` must not be equal to `Source`")
			return
		}
		if err = op.IsWellFormed(); err != nil {
			return
		}
	}

	// TODO check duplication Operations

	err = keypair.MustParse(o.B.Source).Verify(
		base58.Decode(o.H.Hash),
		base58.Decode(o.H.Signature),
	)
	if err != nil {
		return
	}

	return
}

func (o Transaction) Validate() (err error) {
	if o.H.Hash != o.B.GetHashString() {
		err = fmt.Errorf("`Hash` mismatch")
		return
	}

	// TODO check whether `Checkpoint` is in `Block Transaction` and is latest
	// `Checkpoint`
	// TODO check whether `Source` is in `Block Account`
	// TODO check whether the balance of `Source` is greater than `totalAmount`
	/*
		totalAmount := o.GetTotalAmount(true)
	*/

	return
}

func (o Transaction) GetTotalAmount(withFee bool) Amount {
	var amount int64
	for _, op := range o.B.Operations {
		amount += int64(op.B.GetAmount())
	}

	if withFee {
		amount += int64(len(o.B.Operations)) * int64(o.B.Fee)
	}

	return Amount(amount)
}

func (o Transaction) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o Transaction) String() string {
	encoded, _ := json.MarshalIndent(o, "", "  ")
	return string(encoded)
}

func (o *Transaction) Sign(kp keypair.KP) {
	signature, _ := kp.Sign(o.B.GetHash())

	o.H.Signature = base58.Encode(signature)

	return
}

type TransactionHeader struct {
	Version   string `json:"version"`
	Created   string `json:"created"`
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
}

type TransactionBody struct {
	Source     string      `json:"source"`
	Fee        Amount      `json:"fee"`
	Checkpoint string      `json:"checkpoint"`
	Operations []Operation `json:"operations"`
}

func (tb TransactionBody) GetHash() []byte {
	return util.MustGetObjectHash(tb)
}

func (tb TransactionBody) GetHashString() string {
	return base58.Encode(tb.GetHash())
}
