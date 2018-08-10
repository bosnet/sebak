package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"github.com/stellar/go/keypair"
)

func TestMakeHashOfOperationBodyPayment(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Target: kp.Address(),
		Amount: sebakcommon.Amount(100),
	}
	op := Operation{
		H: OperationHeader{Type: OperationPayment},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "8AALKhfgCu2w3ZtbESXHG5ko93Jb1L1yCmFopoJubQh9"
	if hashed != expected {
		t.Errorf("hased != <expected>: '%s' != '%s'", hashed, expected)
	}
}

func TestIsWellFormedOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	if err := op.IsWellFormed(networkID); err != nil {
		t.Errorf("failed to check `Operation.IsWellFormed`: %v", err)
	}
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	obp := TestMakeOperationBodyPayment(0)
	if err := obp.IsWellFormed(networkID); err == nil {
		t.Errorf("failed to `Operation.IsWellFormed`: `Amount` must occur error")
	}
}

func TestSerializeOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	var b []byte
	var err error
	if b, err = op.Serialize(); err != nil {
		t.Errorf("failed to serialize: %v", err)
	} else if len(b) < 1 {
		t.Error("failed to serialize: empty output")
	}

	if _, err = NewOperationFromBytes(b); err != nil {
		t.Errorf("failed to unserialize operation data: %v", err)
	}
}
