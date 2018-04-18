package sebak

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
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
	Sender     string              `json:"sender"`
	Fee        string              `json:"fee"`
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
		Sender:     txt.B.Sender,
		Fee:        txt.B.Fee,
		Checkpoint: txt.B.Checkpoint,
		Operations: operations,
	}

	return
}

func (o Transaction) IsWellFormed() (err error) {
	if _, err = keypair.Parse(o.B.Sender); err != nil {
		err = fmt.Errorf("invalid `Sender`: %v", err)
		return
	}

	var fee int64
	if fee, err = strconv.ParseInt(o.B.Fee, 10, 64); err != nil {
		err = fmt.Errorf("invalid `fee`, '%s'", o.B.Fee)
		return
	}

	if fee < BaseFee {
		err = fmt.Errorf("`fee` must be greater than %d", BaseFee)
		return
	}

	for _, op := range o.B.Operations {
		if ta := op.B.GetTargetAddress(); o.B.Sender == ta {
			err = fmt.Errorf("`Operation.TargetAddress` must be equal to `Sender`")
			return
		}
	}

	err = keypair.MustParse(o.B.Sender).Verify(
		base58.Decode(o.H.Hash),
		base58.Decode(o.H.Signature),
	)
	if err != nil {
		return
	}

	return
}

func (o Transaction) Validate() (err error) {
	var totalAmount int64
	for _, op := range o.B.Operations {
		if err = op.Validate(); err != nil {
			return
		}
		totalAmount += op.B.GetAmount()
	}

	if o.H.Hash != o.B.GetHashString() {
		err = fmt.Errorf("`Hash` mismatch")
		return
	}

	// TODO check whether `Checkpoint` is in `Block Transaction` and is latest
	// `Checkpoint`
	// TODO check whether `Sender` is in `Block Account`
	// TODO check whether the balance of `Sender` is greater than `totalAmount`
	/*
		totalAmount += len(o.B.Operations) * o.B.Fee
	*/

	return
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
	Created   string `json:"created"`
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
}

type TransactionBody struct {
	Sender     string      `json:"sender"`
	Fee        string      `json:"fee"`
	Checkpoint string      `json:"checkpoint"`
	Operations []Operation `json:"operations"`
}

func (tb TransactionBody) GetHash() []byte {
	encoded, _ := GetObjectHash(tb)
	return encoded
}

func (tb TransactionBody) GetHashString() string {
	return base58.Encode(tb.GetHash())
}
