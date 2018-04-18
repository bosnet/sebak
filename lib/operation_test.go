package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
)

func TestGetHashOfOperationBodyPayment(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Receiver: kp.Address(),
		Amount:   "100",
	}
	hashed := opb.GetHashString()

	expected := "2kwCyCVgCztDmWse5oFYMDZhpBYxibejrkqafP4V2xuc"
	if hashed != expected {
		t.Errorf("hased != <expected>: '%s' != '%s'", hashed, expected)
	}
}

func TestIsWellFormedOperation(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Receiver: kp.Address(),
		Amount:   "100",
	}

	op := Operation{
		H: OperationHeader{
			Hash: opb.GetHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}
	if err := op.IsWellFormed(); err != nil {
		t.Errorf("failed to check `Operation.IsWellFormed`: %v", err)
	}
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	kp := keypair.Master("find me")

	obp := OperationBodyPayment{Receiver: kp.Address(), Amount: "00"}
	if err := obp.IsWellFormed(); err == nil {
		t.Errorf("failed to `Operation.IsWellFormed`: `Amount` must occur error")
	}
}

func TestSerializeOperation(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Receiver: kp.Address(),
		Amount:   "10",
	}

	op := Operation{
		H: OperationHeader{
			Hash: opb.GetHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}

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
