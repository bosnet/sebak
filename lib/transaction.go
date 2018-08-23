package sebak

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
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
	Fee        sebakcommon.Amount  `json:"fee"`
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
		Fee:        BaseFee,
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

// Validate checks,
// * source account exists
// * checkpoint is valid
// * source has enough balance to pay
// * and it's `Operations`
func (tx Transaction) Validate(st *sebakstorage.LevelDBBackend) (err error) {
	// check, source exists
	var ba *block.BlockAccount
	if ba, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}

	// check, checkpoint is based on latest checkpoint
	if !tx.IsValidCheckpoint(ba.Checkpoint) {
		err = sebakerror.ErrorTransactionInvalidCheckpoint
		return
	}

	// get the balance at checkpoint
	var bac block.BlockAccountCheckpoint
	bac, err = block.GetBlockAccountCheckpoint(st, tx.B.Source, tx.B.Checkpoint)
	if err != nil {
		return
	}

	totalAmount := tx.TotalAmount(true)

	// check, have enough balance at checkpoint
	if sebakcommon.MustAmountFromString(bac.Balance) < totalAmount {
		err = sebakerror.ErrorTransactionExcessAbilityToPay
		return
	}

	for _, op := range tx.B.Operations {
		if err = op.Validate(st); err != nil {
			return
		}
	}

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

//
// Returns:
//   the total monetary value of this transaction,
//   which is the sum of its operations,
//   optionally with fees
//
// Params:
//   withFee = If fee should be included in the total
//
func (tx Transaction) TotalAmount(withFee bool) sebakcommon.Amount {
	// Note that the transaction shouldn't be constructed invalid
	// (the sum of its Operations should not exceed the maximum supply)
	var amount sebakcommon.Amount
	for _, op := range tx.B.Operations {
		amount = amount.MustAdd(op.B.GetAmount())
	}

	// TODO: This isn't checked anywhere yet
	if withFee {
		amount = amount.MustAdd(tx.B.Fee.MustMult(len(tx.B.Operations)))
	}

	return amount
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
	Source     string             `json:"source"`
	Fee        sebakcommon.Amount `json:"fee"`
	Checkpoint string             `json:"checkpoint"`
	Operations []Operation        `json:"operations"`
}

func (tb TransactionBody) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(tb)
}

func (tb TransactionBody) MakeHashString() string {
	return base58.Encode(tb.MakeHash())
}
