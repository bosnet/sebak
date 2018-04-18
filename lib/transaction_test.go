package sebak

import (
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func TestLoadTransactionFromJSON(t *testing.T) {
	kpReceiver := keypair.Master("find me")
	kpSender := keypair.Master("show me")

	opb := OperationBodyPayment{
		Receiver: kpReceiver.Address(),
		Amount:   "100",
	}

	op := Operation{
		H: OperationHeader{
			Hash: opb.GetHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Sender:     kpSender.Address(),
		Fee:        fmt.Sprintf("%d", BaseFee+10),
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx := Transaction{
		H: TransactionHeader{
			Created: time.Now().Format("2006-01-02T15:04:05+09:00"),
			Hash:    txBody.GetHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSender)

	var b []byte
	var err error
	if b, err = tx.Serialize(); err != nil {
		t.Errorf("failed to serialize transction: %v", err)
	}

	if _, err = NewTransactionFromJSON(b); err != nil {
		t.Errorf("failed to load serialized transction: %v", err)
	}
}

func TestIsWellFormedTransaction(t *testing.T) {
	kpReceiver := keypair.Master("find me")
	kpSender := keypair.Master("show me")

	opb := OperationBodyPayment{
		Receiver: kpReceiver.Address(),
		Amount:   "100",
	}

	op := Operation{
		H: OperationHeader{
			Hash: opb.GetHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Sender:     kpSender.Address(),
		Fee:        fmt.Sprintf("%d", BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx := Transaction{
		H: TransactionHeader{
			Created: time.Now().Format("2006-01-02T15:04:05+09:00"),
			Hash:    txBody.GetHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSender)

	var err error
	if err = tx.Validate(); err != nil {
		t.Errorf("failed to validate transaction: %v", err)
	}
}

func makeTransaction() (tx Transaction) {
	kpReceiver := keypair.Master("find me")
	kpSender := keypair.Master("show me")

	opb := OperationBodyPayment{
		Receiver: kpReceiver.Address(),
		Amount:   "100",
	}

	op := Operation{
		H: OperationHeader{
			Hash: opb.GetHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Sender:     kpSender.Address(),
		Fee:        fmt.Sprintf("%d", BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx = Transaction{
		H: TransactionHeader{
			Created: time.Now().Format("2006-01-02T15:04:05+09:00"),
			Hash:    txBody.GetHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSender)

	return
}

func TestIsWellFormedTransactionWithLowerFee(t *testing.T) {
	var err error

	tx := makeTransaction()
	tx.B.Fee = fmt.Sprintf("%d", BaseFee)
	if err = tx.Validate(); err != nil {
		t.Errorf("transaction must not be failed for fee: %d", BaseFee)
	}
	tx.B.Fee = fmt.Sprintf("%d", BaseFee+1)
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("transaction must not be failed for fee: %d", BaseFee+1)
	}

	tx.B.Fee = fmt.Sprintf("%d", BaseFee-1)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", BaseFee-1)
	}

	tx.B.Fee = fmt.Sprintf("%d", 0)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", 0)
	}
}

func TestIsWellFormedTransactionWithInvalidSenderAddress(t *testing.T) {
	var err error

	tx := makeTransaction()
	tx.B.Sender = "invalid-address"
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for invalid sender: '%s'", tx.B.Sender)
	}
}

func TestIsWellFormedTransactionWithTargetAddressIsSameWithSenderAddress(t *testing.T) {
	var err error

	tx := makeTransaction()
	tx.B.Sender = tx.B.Operations[0].B.GetTargetAddress()
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for same sender: '%s'", tx.B.Sender)
	}
}

func TestIsWellFormedTransactionWithInvalidSignature(t *testing.T) {
	var err error

	tx := makeTransaction()
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("failed to be wellformed for transaction: '%s'", err)
	}

	newSignature, _ := keypair.Master("find me").Sign(tx.B.GetHash())
	tx.H.Signature = base58.Encode(newSignature)

	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for signature verification")
	}
}
