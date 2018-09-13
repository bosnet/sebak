package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/block"
	"github.com/stretchr/testify/require"
)

func TestGetOperationsByTxHashHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, btList, err := prepareTxs(storage, 0, 1, nil)
	require.Nil(t, err)

	bt := btList[0]

	// Do a Request
	url := strings.Replace(GetTransactionOperationsHandlerPattern, "{id}", bt.Hash, -1)
	respBody, err := request(ts, url, false)
	require.Nil(t, err)
	defer respBody.Close()
	reader := bufio.NewReader(respBody)

	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)

	recv := make(map[string]interface{})
	err = json.Unmarshal(readByte, &recv)
	require.Nil(t, err)

	records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

	for _, r := range records {
		item := r.(map[string]interface{})
		hash := item["hash"].(string)
		amount := item["amount"].(string)

		bo, err := block.GetBlockOperation(storage, hash)
		require.Nil(t, err)
		require.NotNil(t, bo)
		require.Equal(t, amount, bo.Amount.String())
	}
}
