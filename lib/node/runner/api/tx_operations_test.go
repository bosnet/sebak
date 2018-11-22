package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
)

func TestGetOperationsByTxHashHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, btList := prepareTxs(storage, 1)
	bt := btList[0]

	{ // unknown transaction
		url := strings.Replace(GetTransactionOperationsHandlerPattern, "{id}", "showme", -1)
		req, _ := http.NewRequest("GET", ts.URL+url, nil)
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	// Do a Request
	url := strings.Replace(GetTransactionOperationsHandlerPattern, "{id}", bt.Hash, -1)
	respBody := request(ts, url, false)
	defer respBody.Close()
	reader := bufio.NewReader(respBody)

	readByte, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	recv := make(map[string]interface{})
	err = json.Unmarshal(readByte, &recv)
	require.NoError(t, err)

	records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

	blk, _ := block.GetBlock(storage, bt.Block)

	for _, r := range records {
		item := r.(map[string]interface{})
		hash := item["hash"].(string)

		bo, err := block.GetBlockOperation(storage, hash)
		require.NoError(t, err)
		require.NotNil(t, bo)
		require.NotNil(t, item["proposed_time"]) // `block.Block.ProposedTime`
		require.Equal(t, blk.ProposedTime, item["proposed_time"].(string))
		require.Equal(t, blk.Confirmed, item["confirmed"].(string))
		require.Equal(t, blk.Height, uint64(item["block_height"].(float64)))
	}
}
