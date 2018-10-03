package common

import (
	"fmt"
	"math"
	"strconv"

	"boscoin.io/sebak/lib/error"
)

// CalculateInflation returns the amount of inflation in every block.
func CalculateInflation(initialBalance Amount, ratio float64) (a Amount, err error) {
	if initialBalance > MaximumBalance {
		err = errors.ErrorMaximumBalanceReached
		return
	}

	if ratio < 0 {
		err = errors.ErrorInvalidInflationRatio
		return
	}

	a = Amount(uint64(math.Round(float64(initialBalance) * ratio)))
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
