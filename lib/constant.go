package sebak

import (
	"regexp"
)

var Version = "0.1+proto"                      // `Version` must follow the SemVer(https://semver.org)
var totalBalance = "1,000,000,000,000.0000000" // 1 trillon with 7 decimal digit
var TotalBalance string
var TotalBalanceLength int

// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
// transaction will be failed to be validated.
var BaseFee int64 = 10000

func init() {
	TotalBalance = regexp.MustCompile("[,\\.]").ReplaceAllString(totalBalance, "")
	TotalBalanceLength = len(TotalBalance)
}
