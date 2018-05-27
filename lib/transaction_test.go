package sebak

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/stellar/go/keypair"
)

func TestLoadTransactionFromJSON(t *testing.T) {
	_, tx := MakeTransactions(1)

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
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	_, tx := MakeTransactions(1)

	var err error
	if err = tx.Validate(st); err != nil {
		t.Errorf("failed to validate transaction: %v", err)
	}
}

func TestIsWellFormedTransactionWithLowerFee(t *testing.T) {
	var err error

	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	kp, tx := MakeTransactions(1)
	tx.B.Fee = Amount(BaseFee)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp)
	if err = tx.Validate(st); err != nil {
		t.Errorf("transaction must not be failed for fee: %d: %v", BaseFee, err)
	}
	tx.B.Fee = Amount(BaseFee + 1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp)
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("transaction must not be failed for fee: %d: %v", BaseFee+1, err)
	}

	tx.B.Fee = Amount(BaseFee - 1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", BaseFee-1)
	}

	tx.B.Fee = Amount(0)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp)
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for fee: %d", 0)
	}
}

func TestIsWellFormedTransactionWithInvalidSourceAddress(t *testing.T) {
	var err error

	_, tx := MakeTransactions(1)
	tx.B.Source = "invalid-address"
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for invalid source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithTargetAddressIsSameWithSourceAddress(t *testing.T) {
	var err error

	_, tx := MakeTransactions(1)
	tx.B.Source = tx.B.Operations[0].B.TargetAddress()
	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for same source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithInvalidSignature(t *testing.T) {
	var err error

	_, tx := MakeTransactions(1)
	if err = tx.IsWellFormed(); err != nil {
		t.Errorf("failed to be wellformed for transaction: '%s'", err)
	}

	newSignature, _ := keypair.Master("find me").Sign([]byte(tx.B.MakeHashString()))
	tx.H.Signature = base58.Encode(newSignature)

	if err = tx.IsWellFormed(); err == nil {
		t.Errorf("transaction must be failed for signature verification")
	}
}
