/**
 * Define the `Amount` type, which is the monetary type used accross the code base
 *
 * One BOSCoin accounts for 10 million currency units.
 * In addition to the `Amount` type, some member functions are defined:
 * - Add / Sub do an addition / substraction and return an error object
 * - AddCheck / SubCheck call `Add` / `Sub` and turn any `error` into a `panic`.
 *   Those are provided for testing / quick prototyping and should not be in production code.
 * - Invariant `panic`s if the instance it's called on violates its invariant (see Contract programming)
 */
package sebak

import (
	"fmt"
	"strconv"

	"github.com/owlchain/sebak/lib/error"
)

const (
	/// 10,000,000 units == 1 BOSCoin
	amountPerCoin Amount = 10000000
	/// The maximum possible supply of coins within any network
	/// It is 10 trillions BOSCoin, or 100,000,000,000,000,000,000 in `Amount`
	MaximumBalance Amount = 1000000000000 * amountPerCoin
	/// An invalid valid, used to make an instance unusable
	invalidValue = Amount(MaximumBalance + 1)
)

/// Main monetary type used accross sebak
type Amount uint64

/// Check this type's invariant, that is, its value is <= MaximumBalance
func (this Amount) Invariant() {
	if this > MaximumBalance {
		// `uint64` is necessary to avoid a recursive call to `String`
		// which would lead to an infinite recursion
		panic(fmt.Errorf("Amount '%d' is higher than the total supply of coins (%d)", uint64(this), uint64(MaximumBalance)))
	}
}

/// Stringer interface implementation
func (a Amount) String() string {
	a.Invariant()
	return strconv.FormatInt(int64(a), 10)
}

/**
 * Add an `Amount` to this `Amount`
 *
 * If the resulting value would overflow maximumAmount, an error is returned,
 * along with the value (which would trigger a `panic` if used).
 */
func (a Amount) Add(added Amount) (n Amount, err error) {
	a.Invariant()
	added.Invariant()
	if n = a + added; n > MaximumBalance {
		err = sebakerror.ErrorMaximumBalanceReached
	}
	return
}

/// Counterpart of `Add` which panic instead of returning an error
/// Useful for debugging and testing, should be avoided in regular code
func (a Amount) MustAdd(added Amount) Amount {
	if v, err := a.Add(added); err != nil {
		panic(err)
	} else {
		return v
	}
}

/**
 * Substract an `Amount` to this `Amount`
 *
 * If the resulting value would underflow, an error is returned,
 * along with an invalid value (which would trigger a `panic` if used).
 */
func (a Amount) Sub(sub Amount) (Amount, error) {
	a.Invariant()
	sub.Invariant()
	if a < sub {
		return invalidValue, sebakerror.ErrorAccountBalanceUnderZero
	}
	return a - sub, nil
}

/// Counterpart of `Sub` which panic instead of returning an error
/// Useful for debugging and testing, should be avoided in regular code
func (a Amount) MustSub(sub Amount) Amount {
	if v, err := a.Sub(sub); err != nil {
		panic(err)
	} else {
		return v
	}
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", strconv.FormatInt(int64(a), 10))), nil
}

func (a *Amount) UnmarshalJSON(b []byte) (err error) {
	var c int64
	if c, err = strconv.ParseInt(string(b[1:len(b)-1]), 10, 64); err != nil {
		return
	}

	*a = Amount(c)

	return
}

func AmountFromBytes(s []byte) (a Amount, err error) {
	var c int64
	if c, err = strconv.ParseInt(string(s), 10, 64); err != nil {
		return
	}

	a = Amount(c)

	return
}

func AmountFromString(s string) (Amount, error) {
	return AmountFromBytes([]byte(s))
}

func MustAmountFromString(s string) Amount {
	a, _ := AmountFromBytes([]byte(s))
	return a
}
