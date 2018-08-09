package sebakcommon

import (
	"strconv"
	"testing"
)

var (
	maximumBalance    = uint64(MaximumBalance)
	maximumBalanceStr = strconv.FormatUint(maximumBalance, 10)
)

func TestAmount_Invariant(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("exceeds max allowable amount value.")
		}
	}()

	amount := Amount(maximumBalance + 1)
	amount.Invariant()
}

func TestAmount_Mult(t *testing.T) {
	if Amount(100).MustMult(50) != Amount(5000) {
		t.Errorf("MustMult returned a wrong result")
	}
	val, err := Amount(100).MultUint(50)
	if err != nil || val != Amount(5000) {
		t.Errorf("MustMult returned an error or a wrong result")
	}
	// Test `MustMult` + overflow failure
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected `panic` did not happen")
			}
		}()
		_ = MaximumBalance.MustMult(2)
		t.Error("Unreachable code")
	}()
	// Test negative value
	_, err = Amount(42).MultInt(-42)
	if err == nil {
		t.Errorf("Expected error on negative value was not triggered")
	}
}

// https://github.com/bosnet/sebak/issues/85
func TestAmount_Uint64OutOfRange(t *testing.T) {
	amount, err := AmountFromString(maximumBalanceStr)

	if amount.String() != maximumBalanceStr {
		t.Errorf("invalid stringified value: %s", amount.String())
	}

	if err != nil {
		t.Errorf("failed to parse number for input string: %s, %v", maximumBalanceStr, err)
	}

	if uint64(amount) != uint64(maximumBalance) {
		t.Errorf("failed to parse number for input string: %s != %s", amount, maximumBalanceStr)
	}

	if data, err := amount.MarshalJSON(); err != nil {
		t.Errorf("marshal error: %v", err)
	} else {
		if string(data)[1:len(data)-1] != maximumBalanceStr {
			t.Errorf("unexpected marshal value. expected: %s, got: %s", maximumBalanceStr, data)
		}

		if err := amount.UnmarshalJSON(data); err != nil {
			t.Errorf("unmarshal error: %v", err)
		}
	}
}
