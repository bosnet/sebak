package sebakcommon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseISO8601(t *testing.T) {
	s := "2018-08-25T14:12:10.090758840+09:00"
	parsed, err := ParseISO8601(s)
	require.Nil(t, err)

	require.Equal(t, parsed.Year(), 2018)
	require.Equal(t, parsed.Month(), time.Month(8))
	require.Equal(t, parsed.Day(), 25)
	require.Equal(t, parsed.Hour(), 14)
	require.Equal(t, parsed.Minute(), 12)
	require.Equal(t, parsed.Second(), 10)
	require.Equal(t, parsed.Nanosecond(), 90758840)

	zone, offset := parsed.Zone()
	require.Equal(t, zone, "KST")
	require.Equal(t, offset, 9*60*60)
}

func TestParseISO8601Timezone(t *testing.T) {
	now := time.Now()

	location, err := time.LoadLocation("Europe/Amsterdam") // Wouter Bothoff lives in Amsterdam.
	require.Nil(t, err)

	nowCEST := now.In(location)
	formattedCEST := FormatISO8601(nowCEST)

	parsed, err := ParseISO8601(formattedCEST)
	require.Nil(t, err)

	require.Equal(t, now.Sub(parsed), time.Duration(0))
}
