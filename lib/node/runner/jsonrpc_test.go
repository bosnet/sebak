package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	rpcjson "github.com/gorilla/rpc/json"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

type jsonrpcServerTestHelper struct {
	server   *httptest.Server
	endpoint *common.Endpoint
	st       *storage.LevelDBBackend
	js       *JSONRPCServer
	t        *testing.T
}

func (jp *jsonrpcServerTestHelper) prepare() {
	jp.server = httptest.NewUnstartedServer(nil)
	endpoint, _ := common.NewEndpointFromString("http://localhost/jsonrpc")
	jp.st = storage.NewTestStorage()

	jp.js = NewJSONRPCServer(endpoint, jp.st)
	jp.server.Config = &http.Server{Handler: jp.js.Ready()}
	jp.server.Start()

	u, _ := url.Parse(jp.server.URL)
	endpoint.Host = u.Host
	endpoint.Scheme = u.Scheme

	jp.endpoint = endpoint

}

func (jp *jsonrpcServerTestHelper) done() {
	jp.server.Close()
	jp.st.Close()
}

func (jp *jsonrpcServerTestHelper) request(method string, args interface{}) *http.Response {
	message, err := rpcjson.EncodeClientRequest(method, &args)
	require.NoError(jp.t, err)

	req, err := http.NewRequest("POST", jp.endpoint.String(), bytes.NewBuffer(message))
	require.NoError(jp.t, err)

	req.Header.Set("Content-Type", "application/json")
	client := new(http.Client)
	resp, err := client.Do(req)
	require.NoError(jp.t, err)
	require.Equal(jp.t, 200, resp.StatusCode)

	return resp
}

func TestJSONRPCServerEcho(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	token := common.NowISO8601()

	args := EchoArgs(token)
	resp := jp.request("Main.Echo", &args)
	defer resp.Body.Close()

	var result EchoResult
	err := rpcjson.DecodeClientResponse(resp.Body, &result)
	require.NoError(t, err)

	require.Equal(t, token, string(result))
}

func TestJSONRPCServerDBHas(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	// store data in storage
	key := "showme"
	jp.st.New(key, key)

	{
		args := DBGetArgs(key)
		resp := jp.request("DB.Has", &args)
		defer resp.Body.Close()

		var result DBHasResult
		err := rpcjson.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, true, bool(result))
	}

	{
		args := DBGetArgs(key + "hahaha")
		resp := jp.request("DB.Has", &args)
		defer resp.Body.Close()

		var result DBHasResult
		err := rpcjson.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, false, bool(result))
	}
}

func TestJSONRPCServerDBGet(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	expected := []string{}
	{ // store data in storage
		total := 30

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%03d", i)
			jp.st.New(key, key)
			expected = append(expected, key)
		}
	}

	for _, exp := range expected {
		args := DBGetArgs(string(exp))
		resp := jp.request("DB.Get", &args)
		defer resp.Body.Close()

		var result DBGetResult
		err := rpcjson.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.Equal(t, exp, string(result.Key))

		var r string
		json.Unmarshal(result.Value, &r)
		require.Equal(t, exp, r)
	}
}

func TestJSONRPCServerDBGetIterator(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	expectedPrefix := string(0x00)
	expected := []string{}
	{ // store data in storage
		total := 10

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%s%03d", expectedPrefix, i)
			err := jp.st.New(key, key)
			require.NoError(t, err)

			expected = append(expected, key)
		}
	}

	{ // store another data, which has different prefix
		prefix := string(0x01)
		total := 3

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%s%03d", prefix, i)
			jp.st.New(key, key)
		}
	}

	{ // with over limit
		args := DBGetIteratorArgs{
			Prefix:  expectedPrefix,
			Options: GetIteratorOptions{Limit: uint64(len(expected) + 100)},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := rpcjson.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, len(expected), len(result.Items))
		for i, item := range result.Items {
			require.Equal(t, expected[i], string(item.Key))
		}
	}

	{ // with reverse
		args := DBGetIteratorArgs{
			Prefix: expectedPrefix,
			Options: GetIteratorOptions{
				Limit:   uint64(len(expected) + 100),
				Reverse: true,
			},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := rpcjson.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, len(expected), len(result.Items))
		for i, item := range result.Items {
			require.Equal(t, expected[len(expected)-i-1], string(item.Key))
		}
	}
}

func TestJSONRPCServerDBGetIteratorWithLimit(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	expectedPrefix := string(0x00)
	expected := []string{}
	{ // store data in storage
		total := 10

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%s%03d", expectedPrefix, i)
			err := jp.st.New(key, key)
			require.NoError(t, err)

			expected = append(expected, key)
		}
	}

	{ // store another data, which has different prefix
		prefix := string(0x01)
		total := 3

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%s%03d", prefix, i)
			jp.st.New(key, key)
		}
	}

	limit := 3
	args := DBGetIteratorArgs{
		Prefix:  expectedPrefix,
		Options: GetIteratorOptions{Limit: uint64(limit)},
	}

	resp := jp.request("DB.GetIterator", args)
	defer resp.Body.Close()

	var result DBGetIteratorResult
	err := rpcjson.DecodeClientResponse(resp.Body, &result)
	require.NoError(t, err)

	require.Equal(t, limit, len(result.Items))
	for i, item := range result.Items {
		require.Equal(t, expected[i], string(item.Key))
	}
}
