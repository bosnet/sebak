package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction"
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

	for _, r := range records {
		item := r.(map[string]interface{})
		hash := item["hash"].(string)

		bo, err := block.GetBlockOperation(storage, hash)
		require.NoError(t, err)
		require.NotNil(t, bo)
	}
}

func TestGetOperationsByTxHashHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp := keypair.Random()
	tx := transaction.TestMakeTransactionWithKeypair(networkID, 10, kp)
	bt := block.NewBlockTransactionFromTransaction("block-hash", 1, common.NowISO8601(), tx)

	boMap := make(map[string]block.BlockOperation)
	for _, op := range tx.B.Operations {
		bo, err := block.NewBlockOperationFromOperation(op, tx, 0)
		require.NoError(t, err)
		boMap[bo.Hash] = bo
	}

	// Wait until request registered to observer
	{
		go func() {
			for {
				observer.BlockOperationObserver.RLock()
				if len(observer.BlockOperationObserver.Callbacks) > 0 {
					observer.BlockOperationObserver.RUnlock()
					break
				}
				observer.BlockOperationObserver.RUnlock()
			}
			for _, bo := range boMap {
				bo.MustSave(storage)
			}
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetTransactionOperationsHandlerPattern, "{id}", bt.Hash, -1)
		respBody := request(ts, url, true)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		// Do stream Request to the Server
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bo := boMap[recv["hash"].(string)]
			r := resource.NewOperation(&bo)
			txS, err := json.Marshal(r.Resource())
			require.NoError(t, err)
			require.Equal(t, txS, line)
		}
	}

	wg.Wait()
}
