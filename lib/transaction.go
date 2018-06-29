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

func (tx Transaction) IsWellFormed(networkID []byte) (err error) {
	// TODO check `Version` format with SemVer

	checker := &TransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{TransactionWellFormedCheckerFuncs},
		NetworkID:      networkID,
		Transaction:    tx,
	}
	if err = sebakcommon.RunChecker(checker, sebakcommon.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (tx Transaction) Validate(st *sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Checkpoint` is in `Block Transaction` and is latest
	// `Checkpoint`
	// TODO check whether `Source` is in `Block Account`
	// TODO check whether the balance of `Source` is greater than `totalAmount`

	return
}

func (tx Transaction) GetType() string {
	return tx.T
}

func (tx Transaction) Equal(m sebakcommon.Message) bool {
	return tx.H.Hash == m.GetHash()
}

func (tx Transaction) IsValidCheckpoint(checkpoint string) bool {
	if tx.B.Checkpoint == checkpoint {
		return true
	}

	var err error
	var inputCheckpoint, currentCheckpoint [2]string
	if inputCheckpoint, err = sebakcommon.ParseCheckpoint(checkpoint); err != nil {
		return false
	}
	if currentCheckpoint, err = sebakcommon.ParseCheckpoint(tx.B.Checkpoint); err != nil {
		return false
	}

	return inputCheckpoint[0] == currentCheckpoint[0]
}

func (tx Transaction) GetHash() string {
	return tx.H.Hash
}

func (tx Transaction) Source() string {
	return tx.B.Source
}

func (tx Transaction) TotalAmount(withFee bool) Amount {
	var amount int64
	for _, op := range tx.B.Operations {
		amount += int64(op.B.GetAmount())
	}

	if withFee {
		amount += int64(len(tx.B.Operations)) * int64(tx.B.Fee)
	}

	return Amount(amount)
}

func (tx Transaction) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(tx)
	return
}

func (tx Transaction) String() string {
	encoded, _ := json.MarshalIndent(tx, "", "  ")
	return string(encoded)
}

func (tx *Transaction) Sign(kp keypair.KP, networkID []byte) {
	tx.H.Hash = tx.B.MakeHashString()
	signature, _ := kp.Sign(append(networkID, []byte(tx.H.Hash)...))

	tx.H.Signature = base58.Encode(signature)

	return
}

// NextSourceCheckpoint generate new checkpoint from current Transaction. It has
// 2 part, "<subtracted>-<added>".
//
// <subtracted>: hash of last paid transaction, it means balance is subtracted
// <added>: hash of last added transaction, it means balance is added
func (tx Transaction) NextSourceCheckpoint() string {
	return sebakcommon.MakeCheckpoint(tx.GetHash(), tx.GetHash())
}

func (tx Transaction) NextTargetCheckpoint() string {
	parsed, _ := sebakcommon.ParseCheckpoint(tx.B.Checkpoint)

	return sebakcommon.MakeCheckpoint(parsed[0], tx.GetHash())
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
