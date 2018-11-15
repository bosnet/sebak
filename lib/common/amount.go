//
// Define the `Amount` type, which is the monetary type used accross the code base
//
// One BOSCoin accounts for 10 million currency units.
// In addition to the `Amount` type, some member functions are defined:
// - `Add` / `Sub` do an addition / substraction and return an error object
// - `MustAdd` / `MustSub` call `Add` / `Sub` and turn any `error` into a `panic`.
//   Those are provided for testing / quick prototyping and should not be in production code.
// - Invariant `panic`s if the instance it's called on violates its invariant (see Contract programming)
//
package common

import (
	"fmt"
	"strconv"

	"boscoin.io/sebak/lib/errors"
	"io"
)

const (
	// 10,000,000 units == 1 BOSCoin
	AmountPerCoin Amount = 10000000
	// The maximum possible supply of coins within any network
	// It is 10 trillions BOSCoin, or 100,000,000,000,000,000,000 in `Amount`
	MaximumBalance Amount = 1000000000000 * AmountPerCoin
	// An invalid valid, used to make an instance unusable
	invalidValue = Amount(MaximumBalance + 1)
	// Amount that can be frozen, currently 10,000 BOS
	// Freezing happens by steps, as one can freeze 10k, 20k, 30k, etc... but not 15k.
	Unit = Amount(10000 * AmountPerCoin)
)

// Main monetary type used accross sebak
type Amount uint64

// Check this type's invariant, that is, its value is <= MaximumBalance
func (this Amount) Invariant() {
	if this > MaximumBalance {
		// `uint64` is necessary to avoid a recursive call to `String`
		// which would lead to an infinite recursion
		panic(fmt.Errorf("Amount '%d' is higher than the total supply of coins (%d)", uint64(this), uint64(MaximumBalance)))
	}
}

func (a Amount) EncodeRLP(w io.Writer) (err error) {
	w.Write([]byte(a.String()))
	return nil
}

// Stringer interface implementation
func (a Amount) String() string {
	a.Invariant()
	return strconv.FormatUint(uint64(a), 10)
}

//
// Add an `Amount` to this `Amount`
//
// If the resulting value would overflow maximumAmount, an error is returned,
// along with the value (which would trigger a `panic` if used).
//
func (a Amount) Add(added Amount) (n Amount, err error) {
	a.Invariant()
	added.Invariant()
	if n = a + added; n > MaximumBalance {
		err = errors.MaximumBalanceReached
	}
	return
}

// Counterpart of `Add` which panic instead of returning an error
// Useful for debugging and testing, should be avoided in regular code
func (a Amount) MustAdd(added Amount) Amount {
	if v, err := a.Add(added); err != nil {
		panic(err)
	} else {
		return v
	}
}

//
// Substract an `Amount` to this `Amount`
//
// If the resulting value would underflow, an error is returned,
// along with an invalid value (which would trigger a `panic` if used).
//
func (a Amount) Sub(sub Amount) (Amount, error) {
	a.Invariant()
	sub.Invariant()
	if a < sub {
		return invalidValue, errors.AccountBalanceUnderZero
	}
	return a - sub, nil
}

//
// Add this `Amount` to itself, `n` times
//
// If the resulting value would overflow maximumAmount, an error is returned,
// along with the value (which would trigger a `panic` if used).
//
func (a Amount) MultInt(n int) (Amount, error) {
	return a.MultInt64(int64(n))
}

/// Ditto
func (a Amount) MultUint(n uint) (Amount, error) {
	return a.MultUint64(uint64(n))
}

/// Ditto
func (a Amount) MultInt64(n int64) (Amount, error) {
	if n < 0 {
		return invalidValue, errors.AccountBalanceUnderZero
	}
	return a.MultUint64(uint64(n))
}

/// Ditto
func (a Amount) MultUint64(n uint64) (Amount, error) {
	if n == 0 {
		return Amount(0), nil
	}

	a.Invariant()
	if uint64(MaximumBalance)/n < uint64(a) {
		return invalidValue, errors.MaximumBalanceReached
	}

	return Amount(uint64(a) * n), nil
}

// Counterpart of `Mult` which panic instead of returning an error
// Useful for debugging and testing, should be avoided in regular code
func (a Amount) MustMult(n int) Amount {
	if v, err := a.MultInt(n); err != nil {
		panic(err)
	} else {
		return v
	}
}

// Counterpart of `Sub` which panic instead of returning an error
// Useful for debugging and testing, should be avoided in regular code
func (a Amount) MustSub(sub Amount) Amount {
	if v, err := a.Sub(sub); err != nil {
		panic(err)
	} else {
		return v
	}
}

// Implement JSON's Marshaler interface
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", a.String())), nil
}

// Implement JSON's Unmarshaler interface
// If Unmarshalling errors, `a` will have an `invalidValue`
func (a *Amount) UnmarshalJSON(b []byte) (err error) {
	*a, err = AmountFromString(string(b[1 : len(b)-1]))
	return
}

// Parse an `Amount` from a string input
//
// Params:
//   str = a string consisting only of numbers, expressing an amount in GON
//
// Returns:
//  A valid `Amount` and a `nil` error, or an invalid amount and an `error`
func AmountFromString(str string) (Amount, error) {
	if value, err := strconv.ParseUint(str, 10, 64); err != nil {
		return invalidValue, err
	} else {
		return Amount(value), nil
	}
}

// Same as AmountFromString, except it `panic`s if an error happens
func MustAmountFromString(str string) Amount {
	if value, err := AmountFromString(str); err != nil {
		panic(err)
	} else {
		return value
	}
}
