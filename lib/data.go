package sebak

import (
	"fmt"
	"strconv"
)

type Amount uint64

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
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
