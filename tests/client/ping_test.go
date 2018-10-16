package client

import (
	"boscoin.io/sebak/lib/client"
	"fmt"
	"net/http"
	"testing"
)

func TestPing(t *testing.T) {
	c := client.NewClient("https://127.0.0.1:2830")
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	a, e := c.LoadAccount("GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ")
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(a)
}
