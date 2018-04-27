package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
)

func TestMakeHashOfOperationBodyPayment(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Target: kp.Address(),
		Amount: Amount(100),
	}
	hashed := opb.MakeHashString()

	expected := "CWLnH1pNLTehPpoid13j81vX7rSV6FhDXqqhqKHZhLoH"
	if hashed != expected {
		t.Errorf("hased != <expected>: '%s' != '%s'", hashed, expected)
	}
}

func TestIsWellFormedOperation(t *testing.T) {
	op := MakeOperation()
	if err := op.IsWellFormed(); err != nil {
		t.Errorf("failed to check `Operation.IsWellFormed`: %v", err)
	}
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	obp := MakeOperationBodyPayment()
	obp.Amount = 0
	if err := obp.IsWellFormed(); err == nil {
		t.Errorf("failed to `Operation.IsWellFormed`: `Amount` must occur error")
	}
}

func TestSerializeOperation(t *testing.T) {
	op := MakeOperation()
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
