package sebak

import (
	"regexp"
)

// Version is Top-level of version. It must follow the SemVer(https://semver.org)
var Version = "0.1+proto"

// TotalBalanceWithComma is the maximum currency limit, you can not make the currency
// over `TotalBalanceWithComma'. The default is 1 trillon with 7 decimal digit.
var TotalBalanceWithComma = "1,000,000,000,000.0000000"

var TotalBalance string
var TotalBalanceLength int

// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
// transaction will be failed to be validated.
var BaseFee int64 = 10000

func init() {
	TotalBalance = regexp.MustCompile("[,\\.]").ReplaceAllString(TotalBalanceWithComma, "")
	TotalBalanceLength = len(TotalBalance)
}
