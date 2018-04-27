package sebak

import (
	"regexp"
)

// Top-level of version. It must follow the SemVer(https://semver.org)
var Version = "0.1+proto"

// totalBalance is the maximum currency limit, you can not make the currency
// over `totalBalance'. The default is 1 trillon with 7 decimal digit.
var totalBalance = "1,000,000,000,000.0000000"

var TotalBalance string
var TotalBalanceLength int

// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
// transaction will be failed to be validated.
var BaseFee int64 = 10000

func init() {
	TotalBalance = regexp.MustCompile("[,\\.]").ReplaceAllString(totalBalance, "")
	TotalBalanceLength = len(TotalBalance)
}
