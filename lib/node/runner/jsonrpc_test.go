package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jsonrpc "github.com/gorilla/rpc/json"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
)

type jsonrpcServerTestHelper struct {
	server   *httptest.Server
	endpoint *common.Endpoint
	st       *storage.LevelDBBackend
	js       *jsonrpcServer
	t        *testing.T
}

func (jp *jsonrpcServerTestHelper) prepare() {
	jp.server = httptest.NewUnstartedServer(nil)
	endpoint, _ := common.NewEndpointFromString("http://localhost/jsonrpc")
	jp.st = storage.NewTestStorage()

	jp.js = newJSONRPCServer(endpoint, jp.st)
	jp.server.Config = &http.Server{Handler: jp.js.Ready()}
	jp.server.Start()
	jp.js.app.snapshots.start()

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
	message, err := jsonrpc.EncodeClientRequest(method, &args)
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

func TestJSONRPCServerDBEcho(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	token := common.NowISO8601()

	args := DBEchoArgs(token)
	resp := jp.request("DB.Echo", &args)
	defer resp.Body.Close()

	var result DBEchoResult
	err := jsonrpc.DecodeClientResponse(resp.Body, &result)
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

	{ // without snapshot
		args := &DBHasArgs{Key: key}
		resp := jp.request("DB.Has", args)
		defer resp.Body.Close()

		var result DBHasResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	{ // wrong snapshot
		args := &DBHasArgs{Snapshot: "findme", Key: key}
		resp := jp.request("DB.Has", args)
		defer resp.Body.Close()

		var result DBHasResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	var snapshot string
	{ // OpenSnapshot
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()

		var result DBOpenSnapshotResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.NotEmpty(t, result.Snapshot)

		snapshot = result.Snapshot
	}

	{
		args := &DBGetArgs{Snapshot: snapshot, Key: key}
		resp := jp.request("DB.Has", args)
		defer resp.Body.Close()

		var result DBHasResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, true, bool(result))
	}

	{
		args := &DBGetArgs{Snapshot: snapshot, Key: key + "hahaha"}
		resp := jp.request("DB.Has", &args)
		defer resp.Body.Close()

		var result DBHasResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, false, bool(result))
	}

	{ // ReleaseSnapshot
		{
			resp := jp.request("DB.ReleaseSnapshot", &DBReleaseSnapshot{Snapshot: snapshot})
			defer resp.Body.Close()

			var result DBReleaseSnapshotResult
			jsonrpc.DecodeClientResponse(resp.Body, &result)
			require.True(t, bool(result))
		}

		resp := jp.request("DB.Has", DBGetArgs{Snapshot: snapshot, Key: key})
		defer resp.Body.Close()

		var result DBHasResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
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

	{ // without snapshot
		args := &DBGetArgs{Key: expected[0]}
		resp := jp.request("DB.Get", args)
		defer resp.Body.Close()

		var result DBGetResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	{ // wrong snapshot
		args := &DBGetArgs{Snapshot: "findme", Key: expected[0]}
		resp := jp.request("DB.Get", args)
		defer resp.Body.Close()

		var result DBGetResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	var snapshot string
	{ // OpenSnapshot
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()

		var result DBOpenSnapshotResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.NotEmpty(t, result.Snapshot)

		snapshot = result.Snapshot
	}

	for _, exp := range expected {
		args := DBGetArgs{Snapshot: snapshot, Key: exp}
		resp := jp.request("DB.Get", &args)
		defer resp.Body.Close()

		var result DBGetResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.Equal(t, exp, string(result.Key))

		var r string
		json.Unmarshal(result.Value, &r)
		require.Equal(t, exp, r)
	}

	{ // ReleaseSnapshot
		{
			resp := jp.request("DB.ReleaseSnapshot", &DBReleaseSnapshot{Snapshot: snapshot})
			defer resp.Body.Close()

			var result DBReleaseSnapshotResult
			jsonrpc.DecodeClientResponse(resp.Body, &result)
			require.True(t, bool(result))
		}

		for _, exp := range expected {
			args := DBGetArgs{Snapshot: snapshot, Key: exp}
			resp := jp.request("DB.Get", &args)
			defer resp.Body.Close()

			var result DBGetResult
			err := jsonrpc.DecodeClientResponse(resp.Body, &result)
			require.Error(t, err, errors.SnapshotNotFound.Error())
		}
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

	var snapshot string
	{ // OpenSnapshot
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()

		var result DBOpenSnapshotResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.NotEmpty(t, result.Snapshot)

		snapshot = result.Snapshot
	}

	{ // with over limit
		args := DBGetIteratorArgs{
			Snapshot: snapshot,
			Prefix:   expectedPrefix,
			Options:  GetIteratorOptions{Limit: uint64(len(expected) + 100)},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, len(expected), len(result.Items))
		for i, item := range result.Items {
			require.Equal(t, expected[i], string(item.Key))
		}
	}

	{ // with reverse
		args := DBGetIteratorArgs{
			Snapshot: snapshot,
			Prefix:   expectedPrefix,
			Options: GetIteratorOptions{
				Limit:   uint64(len(expected) + 100),
				Reverse: true,
			},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
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

	var snapshot string
	{ // OpenSnapshot
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()

		var result DBOpenSnapshotResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.NotEmpty(t, result.Snapshot)

		snapshot = result.Snapshot
	}

	limit := 3
	args := DBGetIteratorArgs{
		Prefix:   expectedPrefix,
		Snapshot: snapshot,
		Options:  GetIteratorOptions{Limit: uint64(limit)},
	}

	resp := jp.request("DB.GetIterator", args)
	defer resp.Body.Close()

	var result DBGetIteratorResult
	err := jsonrpc.DecodeClientResponse(resp.Body, &result)
	require.NoError(t, err)

	require.Equal(t, limit, len(result.Items))
	for i, item := range result.Items {
		require.Equal(t, expected[i], string(item.Key))
	}
}

func TestJSONRPCServerDBSnapshot(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	expectedPrefix := string(0x00)

	{ // without snapshot
		args := DBGetIteratorArgs{
			Prefix:  expectedPrefix,
			Options: GetIteratorOptions{Limit: 100},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	{ // wrong snapshot
		args := DBGetIteratorArgs{
			Snapshot: "showme",
			Prefix:   expectedPrefix,
			Options:  GetIteratorOptions{Limit: 100},
		}
		resp := jp.request("DB.GetIterator", &args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
	}

	var snapshot string
	{ // OpenSnapshot
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()

		var result DBOpenSnapshotResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)
		require.NotEmpty(t, result.Snapshot)

		snapshot = result.Snapshot
	}

	{ // empty result
		args := DBGetIteratorArgs{
			Prefix:   expectedPrefix,
			Snapshot: snapshot,
			Options:  GetIteratorOptions{Limit: 100},
		}

		resp := jp.request("DB.GetIterator", args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, 0, len(result.Items))
	}

	total := 10
	{ // store data, but snapshot does not know

		for i := 0; i < total; i++ {
			key := fmt.Sprintf("%s%03d", expectedPrefix, i)
			err := jp.st.New(key, key)
			require.NoError(t, err)
		}

		args := DBGetIteratorArgs{
			Prefix:   expectedPrefix,
			Snapshot: snapshot,
			Options:  GetIteratorOptions{Limit: 100},
		}

		resp := jp.request("DB.GetIterator", args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.NoError(t, err)

		require.Equal(t, 0, len(result.Items))
	}

	{ // with new snapshot, the result will be found
		var snapshot string
		{
			resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
			defer resp.Body.Close()

			var result DBOpenSnapshotResult
			jsonrpc.DecodeClientResponse(resp.Body, &result)
			snapshot = result.Snapshot
		}

		args := DBGetIteratorArgs{
			Snapshot: snapshot,
			Prefix:   expectedPrefix,
			Options:  GetIteratorOptions{Limit: 100},
		}

		resp := jp.request("DB.GetIterator", args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Equal(t, total, len(result.Items))
	}

	{ // after ReleaseSnapshot, the request will be failed
		var snapshot string
		{ // OpenSnapshot
			resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
			defer resp.Body.Close()

			var result DBOpenSnapshotResult
			jsonrpc.DecodeClientResponse(resp.Body, &result)
			snapshot = result.Snapshot
		}

		{ // ReleaseSnapshot
			resp := jp.request("DB.ReleaseSnapshot", &DBReleaseSnapshot{Snapshot: snapshot})
			defer resp.Body.Close()

			var result DBReleaseSnapshotResult
			jsonrpc.DecodeClientResponse(resp.Body, &result)
			require.True(t, bool(result))
		}

		args := DBGetIteratorArgs{
			Snapshot: snapshot,
			Prefix:   expectedPrefix,
			Options:  GetIteratorOptions{Limit: 100},
		}

		resp := jp.request("DB.GetIterator", args)
		defer resp.Body.Close()

		var result DBGetIteratorResult
		err := jsonrpc.DecodeClientResponse(resp.Body, &result)
		require.Error(t, err, errors.SnapshotNotFound.Error())
		require.Equal(t, 0, len(result.Items))
	}
}

func TestJSONRPCServerDBSnapshotMaxSnapshotsReached(t *testing.T) {
	jp := jsonrpcServerTestHelper{t: t}
	jp.prepare()
	defer jp.done()

	maxSnapshots := uint64(3)
	jp.js.app.snapshots.maxSnapshots = maxSnapshots

	for i := uint64(0); i < maxSnapshots; i++ {
		resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
		defer resp.Body.Close()
	}

	resp := jp.request("DB.OpenSnapshot", &DBOpenSnapshotResult{})
	defer resp.Body.Close()

	var result DBOpenSnapshotResult
	err := jsonrpc.DecodeClientResponse(resp.Body, &result)
	require.Error(t, err, errors.SnapshotLimitReached.Error())
}
