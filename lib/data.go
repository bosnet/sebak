package sebak

import (
	"fmt"
	"strconv"

	"github.com/owlchain/sebak/lib/error"
)

type Amount uint64

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) Add(i int64) (n Amount, err error) {
	b := int64(a)
	if b+i < 0 {
		err = sebakerror.ErrorAccountBalanceUnderZero
		return
	}

	n = Amount(b + i)
	return
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
