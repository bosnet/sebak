package sebak

import (
	"context"
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

type Transaction struct {
	T string
	H TransactionHeader
	B TransactionBody
}

type TransactionFromJSON struct {
	T string
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

	tx.T = txt.T
	tx.H = txt.H
	tx.B = TransactionBody{
		Source:     txt.B.Source,
		Fee:        txt.B.Fee,
		Checkpoint: txt.B.Checkpoint,
		Operations: operations,
	}

	return
}

var TransactionWellFormedCheckerFuncs = []util.CheckerFunc{
	checkTransactionSource,
	checkTransactionBaseFee,
	checkTransactionOperationIsWellFormed,
	checkTransactionVerifySignature,
	checkTransactionHashMatch,
}

func (o Transaction) IsWellFormed() (err error) {
	if _, err = util.Checker(context.Background(), TransactionWellFormedCheckerFuncs...)(o); err != nil {
		return
	}

	// TODO check `Version` format with SemVer
	// TODO check duplication Operations
	return
}

func (o Transaction) Validate(st *storage.LevelDBBackend) (err error) {
	// TODO check whether `Checkpoint` is in `Block Transaction` and is latest
	// `Checkpoint`
	// TODO check whether `Source` is in `Block Account`
	// TODO check whether the balance of `Source` is greater than `totalAmount`
	/*
		totalAmount := o.GetTotalAmount(true)
	*/

	return
}

func (o Transaction) GetType() string {
	return o.T
}

func (o Transaction) Equal(m util.Message) bool {
	return o.H.Hash == m.GetHash()
}

func (o Transaction) GetHash() string {
	return o.H.Hash
}

func (o Transaction) TotalAmount(withFee bool) Amount {
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
	signature, _ := kp.Sign([]byte(o.H.Hash))

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

func (tb TransactionBody) MakeHash() []byte {
	return util.MustMakeObjectHash(tb)
}

func (tb TransactionBody) MakeHashString() string {
	return base58.Encode(tb.MakeHash())
}
