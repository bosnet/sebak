package sebak

import (
	"fmt"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
)

func TestLoadTransactionFromJSON(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)

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
	_, tx := TestMakeTransaction(networkID, 1)

	var err error
	if err = tx.Validate(st); err != nil {
		t.Errorf("failed to validate transaction: %v", err)
	}
}

func TestIsWellFormedTransactionWithLowerFee(t *testing.T) {
	var err error

	kp, tx := TestMakeTransaction(networkID, 1)
	tx.B.Fee = BaseFee
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	if err = tx.IsWellFormed(networkID); err != nil {
		t.Errorf("transaction must not be failed for fee: %d: %v", BaseFee, err)
	}
	tx.B.Fee = BaseFee.MustAdd(1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	if err = tx.IsWellFormed(networkID); err != nil {
		t.Errorf("transaction must not be failed for fee: %d: %v", BaseFee+1, err)
	}

	tx.B.Fee = BaseFee.MustSub(1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	if err = tx.IsWellFormed(networkID); err == nil {
		t.Errorf("transaction must be failed for fee: %d", BaseFee-1)
	}

	tx.B.Fee = sebakcommon.Amount(0)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	if err = tx.IsWellFormed(networkID); err == nil {
		t.Errorf("transaction must be failed for fee: %d", 0)
	}
}

func TestIsWellFormedTransactionWithInvalidSourceAddress(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	tx.B.Source = "invalid-address"
	if err = tx.IsWellFormed(networkID); err == nil {
		t.Errorf("transaction must be failed for invalid source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithTargetAddressIsSameWithSourceAddress(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	tx.B.Source = tx.B.Operations[0].B.TargetAddress()
	if err = tx.IsWellFormed(networkID); err == nil {
		t.Errorf("transaction must be failed for same source: '%s'", tx.B.Source)
	}
}

func TestIsWellFormedTransactionWithInvalidSignature(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	if err = tx.IsWellFormed(networkID); err != nil {
		t.Errorf("failed to be wellformed for transaction: '%s'", err)
	}

	newSignature, _ := keypair.Master("find me").Sign(append(networkID, []byte(tx.B.MakeHashString())...))
	tx.H.Signature = base58.Encode(newSignature)

	if err = tx.IsWellFormed(networkID); err == nil {
		t.Errorf("transaction must be failed for signature verification")
	}
}

func TestTransactionIsValidCheckpoint(t *testing.T) {
	networkID := []byte("hehe")
	kpSource, _ := keypair.Random()

	tx := TestMakeTransactionWithKeypair(networkID, 1, kpSource)
	l := strings.SplitN(tx.B.Checkpoint, "-", 2)

	newCheckpoint := fmt.Sprintf("%s-%s", l[0], TestGenerateNewCheckpoint())
	if !(tx.IsValidCheckpoint(tx.B.Checkpoint)) {
		t.Error("checkpoint has same with tx")
		return
	}
	if !tx.IsValidCheckpoint(newCheckpoint) {
		t.Error("checkpoint has same head with tx")
		return
	}
}
