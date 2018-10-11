package network

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"boscoin.io/sebak/lib/common"
	"github.com/stretchr/testify/require"
)

func TestRecoverMiddleware(t *testing.T) {
	endpoint, err := common.NewEndpointFromString(
		fmt.Sprintf("http://localhost:%s", getPort()),
	)
	require.Nil(t, err)

	network, err := makeTestHTTP2NetworkForTLS(endpoint)
	require.Nil(t, err)
	defer network.Stop()

	handlerURL := UrlPathPrefixAPI + "/test"
	panicMsg := "Don't panic,just use go"
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(panicMsg)
	}

	VerboseLogs = false
	network.AddMiddleware(RouterNameAPI, RecoverMiddleware(nil))
	network.AddHandler(handlerURL, handler)

	{
		// with normal HTTP2Client
		client, err := common.NewHTTP2Client(
			defaultTimeout,
			defaultIdleTimeout,
			false,
		)
		require.Nil(t, err)

		resp, err := client.Get(endpoint.String()+handlerURL, http.Header{})
		require.Nil(t, err)
		require.Equal(t, 500, resp.StatusCode)
		require.Equal(t, "application/problem+json", resp.Header["Content-Type"][0])

		bs, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		require.Nil(t, err)

		var msg map[string]interface{}
		err = json.Unmarshal(bs, &msg)
		require.Nil(t, err)
		require.Equal(t, "panic: "+panicMsg, msg["title"])
	}
}
