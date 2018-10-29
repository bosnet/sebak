package common

import (
	"fmt"
	"math"
	"strconv"

	"boscoin.io/sebak/lib/errors"
)

// CalculateInflation returns the amount of inflation in every block.
func CalculateInflation(initialBalance Amount) (a Amount, err error) {
	if initialBalance > MaximumBalance {
		err = errors.MaximumBalanceReached
		return
	}

	a = Amount(uint64(math.Round(float64(initialBalance) * InflationRatio)))
	return
}

func InflationRatio2String(ratio float64) string {
	return fmt.Sprintf("%.17f", ratio)
}

func String2InflationRatio(s string) (ratio float64, err error) {
	if ratio, err = strconv.ParseFloat(s, 64); err != nil {
		return
	}

	return
}
