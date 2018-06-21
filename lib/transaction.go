package sebak

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

// TODO versioning

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

func NewTransaction(source, checkpoint string, ops ...Operation) (tx Transaction, err error) {
	if len(ops) < 1 {
		err = sebakerror.ErrorTransactionEmptyOperations
		return
	}

	txBody := TransactionBody{
		Source:     source,
		Fee:        Amount(BaseFee),
		Checkpoint: checkpoint,
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	return
}

var TransactionWellFormedCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckTransactionCheckpoint,
	CheckTransactionSource,
	CheckTransactionBaseFee,
	CheckTransactionOperation,
	CheckTransactionVerifySignature,
	CheckTransactionHashMatch,
}

func (o Transaction) IsWellFormed(networkID []byte) (err error) {
	// TODO check `Version` format with SemVer

	checker := &TransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{TransactionWellFormedCheckerFuncs},
		NetworkID:      networkID,
		Transaction:    o,
	}
	if err = sebakcommon.RunChecker(checker, sebakcommon.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (o Transaction) Validate(st *sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Checkpoint` is in `Block Transaction` and is latest
	// `Checkpoint`
	// TODO check whether `Source` is in `Block Account`
	// TODO check whether the balance of `Source` is greater than `totalAmount`

	return
}

func (o Transaction) GetType() string {
	return o.T
}

func (o Transaction) Equal(m sebakcommon.Message) bool {
	return o.H.Hash == m.GetHash()
}

func (o Transaction) IsValidCheckpoint(checkpoint string) bool {
	if o.B.Checkpoint == checkpoint {
		return true
	}

	var err error
	var inputCheckpoint, currentCheckpoint [2]string
	if inputCheckpoint, err = sebakcommon.ParseCheckpoint(checkpoint); err != nil {
		return false
	}
	if currentCheckpoint, err = sebakcommon.ParseCheckpoint(o.B.Checkpoint); err != nil {
		return false
	}

	return inputCheckpoint[0] == currentCheckpoint[0]
}

func (o Transaction) GetHash() string {
	return o.H.Hash
}

func (o Transaction) Source() string {
	return o.B.Source
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

func (o *Transaction) Sign(kp keypair.KP, networkID []byte) {
	o.H.Hash = o.B.MakeHashString()
	signature, _ := kp.Sign(append(networkID, []byte(o.H.Hash)...))

	o.H.Signature = base58.Encode(signature)

	return
}

// NextSourceCheckpoint generate new checkpoint from current Transaction. It has
// 2 part, "<subtracted>-<added>".
//
// <subtracted>: hash of last paid transaction, it means balance is subtracted
// <added>: hash of last added transaction, it means balance is added
func (o Transaction) NextSourceCheckpoint() string {
	return sebakcommon.MakeCheckpoint(o.GetHash(), o.GetHash())
}

func (o Transaction) NextTargetCheckpoint() string {
	parsed, _ := sebakcommon.ParseCheckpoint(o.B.Checkpoint)

	return sebakcommon.MakeCheckpoint(parsed[0], o.GetHash())
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
	return sebakcommon.MustMakeObjectHash(tb)
}

func (tb TransactionBody) MakeHashString() string {
	return base58.Encode(tb.MakeHash())
}

func FinishTransaction(st *sebakstorage.LevelDBBackend, ballot Ballot, tx Transaction) (err error) {
	var raw []byte
	raw, err = ballot.Data().Serialize()
	if err != nil {
		return
	}

	var ts *sebakstorage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	bt := NewBlockTransactionFromTransaction(tx, raw)
	if err = bt.Save(ts); err != nil {
		ts.Discard()
		return
	}
	for _, op := range tx.B.Operations {
		if err = FinishOperation(ts, tx, op); err != nil {
			ts.Discard()
			return
		}
	}

	var baSource *BlockAccount
	if baSource, err = GetBlockAccount(ts, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		ts.Discard()
		return
	}

	if err = baSource.Withdraw(tx.TotalAmount(true), tx.NextSourceCheckpoint()); err != nil {
		ts.Discard()
		return
	}

	if err = baSource.Save(ts); err != nil {
		ts.Discard()
		return
	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}
