package common

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"boscoin.io/sebak/lib/errors"
)

var (
	TrueQueryStringValue  []string = []string{"true", "yes", "1"}
	FalseQueryStringValue []string = []string{"false", "no", "0"}
)

// ParseBoolQueryString will parse boolean value from url.Value.
// By default, `Reverse` is `false`. If 'true', '1', 'yes', it will be `true`
// If 'false', '0', 'no', it will be `false`
// If not `true` nor `false, `errors.InvalidQueryString` will be occurred.
func ParseBoolQueryString(v string) (yesno bool, err error) {
	if _, yesno = InStringArray(TrueQueryStringValue, strings.ToLower(v)); yesno {
		return
	}
	if _, ok := InStringArray(FalseQueryStringValue, strings.ToLower(v)); ok {
		yesno = false
		return
	}

	err = errors.InvalidQueryString
	return
}

func StrictURLParse(s string) (u *url.URL, err error) {
	u, err = url.Parse(s)
	if err != nil {
		return
	}

	if len(u.Scheme) < 1 {
		err = fmt.Errorf("empty `u.Scheme`")
		return
	}

	if len(u.Host) < 1 {
		err = fmt.Errorf("empty `u.Host`")
		return
	}

	if len(u.Host) < 1 {
		err = fmt.Errorf("empty `u.Host`")
		return
	}

	if !strings.Contains(u.Host, ":") {
		return
	}

	var host, port string
	if host, port, err = net.SplitHostPort(u.Host); err != nil {
		return
	} else if len(host) < 1 {
		err = fmt.Errorf("empty `host`")
		return
	} else if len(port) > 0 {
		if _, err = strconv.ParseUint(port, 10, 64); err != nil {
			err = fmt.Errorf("invalid `port`")
			return
		}
	}

	return
}
