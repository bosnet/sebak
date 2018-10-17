package client

import (
	"fmt"
	"net/http"
	"testing"

	"boscoin.io/sebak/lib/client"
	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	_, e := c.LoadAccount("GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ")
	require.Nil(t, e)
}
