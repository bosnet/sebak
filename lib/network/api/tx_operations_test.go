package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stellar/go/keypair"
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

		bo, err := block.GetBlockOperation(storage, hash)
		require.Nil(t, err)
		require.NotNil(t, bo)
	}
}

func TestGetOperationsByTxHashHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	tx := transaction.TestMakeTransactionWithKeypair(networkID, 10, kp)
	bt := block.NewBlockTransactionFromTransaction("block-hash", 1, tx, nil)

	boMap := make(map[string]block.BlockOperation)
	for _, op := range tx.B.Operations {
		bo := block.NewBlockOperationFromOperation(op, tx, 0)
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
				bo.Save(storage)
			}
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetTransactionOperationsHandlerPattern, "{id}", bt.Hash, -1)
		respBody, err := request(ts, url, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		// Do stream Request to the Server
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bo := boMap[recv["hash"].(string)]
			r := resource.NewOperation(&bo)
			txS, err := json.Marshal(r.Resource())
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}
	}

	wg.Wait()
}
