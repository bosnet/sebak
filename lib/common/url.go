package common

import (
	"strings"

	"boscoin.io/sebak/lib/error"
)

var (
	TrueQueryStringValue  []string = []string{"true", "yes", "1"}
	FalseQueryStringValue []string = []string{"false", "no", "0"}
)

// ParseBoolQueryString will parse boolean value from url.Value.
// By default, `Reverse` is `false`. If 'true', '1', 'yes', it will be `true`
// If 'false', '0', 'no', it will be `false`
// If not `true` nor `false, `errors.ErrorInvalidQueryString` will be occurred.
func ParseBoolQueryString(v string) (yesno bool, err error) {
	if _, yesno = InStringArray(TrueQueryStringValue, strings.ToLower(v)); yesno {
		return
	}
	if _, ok := InStringArray(FalseQueryStringValue, strings.ToLower(v)); ok {
		yesno = false
		return
	}

	err = errors.ErrorInvalidQueryString
	return
}
