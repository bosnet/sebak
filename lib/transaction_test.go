package sebak

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/stellar/go/keypair"
)

func TestLoadTransactionFromJSON(t *testing.T) {
	tx := MakeTransaction(1)

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
	st, _ := storage.NewTestMemoryLevelDBBackend()
	tx := MakeTransaction(1)

	var err error
	if err = tx.Validate(st); err != nil {
		t.Errorf("failed to validate transaction: %v", err)
	}
}

func TestIsWellFormedTransactionWithLowerFee(t *testing.T) {
	var err error

	st, _ := storage.NewTestMemoryLevelDBBackend()
	tx := MakeTransaction(1)
	tx.B.Fee = Amount(BaseFee)
	if err = tx.Validate(st); err != nil {
		t.Errorf("transaction must not be failed for fee: %d", BaseFee)
	}
	tx.B.Fee = Amount(BaseFee + 1)
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("transaction must not be failed for fee: %d", BaseFee+1)
	}

	tx.B.Fee = Amount(BaseFee - 1)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", BaseFee-1)
	}

	tx.B.Fee = Amount(0)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", 0)
	}
}

func TestIsWellFormedTransactionWithInvalidSourceAddress(t *testing.T) {
	var err error

	tx := MakeTransaction(1)
	tx.B.Source = "invalid-address"
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for invalid source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithTargetAddressIsSameWithSourceAddress(t *testing.T) {
	var err error

	tx := MakeTransaction(1)
	tx.B.Source = tx.B.Operations[0].B.GetTargetAddress()
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for same source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithInvalidSignature(t *testing.T) {
	var err error

	tx := MakeTransaction(1)
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("failed to be wellformed for transaction: '%s'", err)
	}

	newSignature, _ := keypair.Master("find me").Sign([]byte(tx.B.MakeHashString()))
	tx.H.Signature = base58.Encode(newSignature)

	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for signature verification")
	}
}
