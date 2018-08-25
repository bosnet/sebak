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

	require.Equal(t, 2018, parsed.Year())
	require.Equal(t, time.Month(8), parsed.Month())
	require.Equal(t, 25, parsed.Day())
	require.Equal(t, 14, parsed.Hour())
	require.Equal(t, 12, parsed.Minute())
	require.Equal(t, 10, parsed.Second())
	require.Equal(t, 90758840, parsed.Nanosecond())

	_, offset := parsed.Zone()
	require.Equal(t, 9*60*60, offset)
}

func TestParseISO8601Timezone(t *testing.T) {
	now := time.Now()

	location, err := time.LoadLocation("Europe/Amsterdam") // Wouter Bothoff lives in Amsterdam.
	require.Nil(t, err)

	nowCEST := now.In(location)
	formattedCEST := FormatISO8601(nowCEST)

	parsed, err := ParseISO8601(formattedCEST)
	require.Nil(t, err)

	require.Equal(t, time.Duration(0), now.Sub(parsed))
}
