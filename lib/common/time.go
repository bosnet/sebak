package sebakcommon

import "time"

const (
	TIMEFORMAT_ISO8601 string = "2006-01-02T15:04:05.000000000Z07:00"
)

func FormatISO8601(t time.Time) string {
	return t.Format(TIMEFORMAT_ISO8601)
}

func NowISO8601() string {
	return FormatISO8601(time.Now())
}

func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(TIMEFORMAT_ISO8601, s)
}
