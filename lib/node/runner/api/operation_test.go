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
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction/operation"
)

func TestGetOperationsByAccountHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, boList := prepareOps(storage, 10)

	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	{
		// unknown address
		req, _ := http.NewRequest("GET", ts.URL+url, nil)
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	{
		ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}

	{
		// Do a Request
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")

			blk, _ := block.GetBlockByHeight(storage, uint64(bt["block_height"].(float64)))
			require.Equal(t, blk.ProposedTime, bt["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, bt["confirmed"].(string))
		}
	}
}

func TestGetOperationsByAccountHandlerWithType(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, boList := prepareOps(storage, 10)
	ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
	ba.MustSave(storage)

	// Do a Request
	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	{
		url := url + "?type=" + string(operation.TypeCreateAccount)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"]
		require.Nil(t, records)
	}

	{
		url := url + "?type=" + string(operation.TypePayment)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")

			blk, _ := block.GetBlockByHeight(storage, uint64(bt["block_height"].(float64)))
			require.Equal(t, blk.ProposedTime, bt["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, bt["confirmed"].(string))
		}
	}

}

func TestGetOperationsByAccountHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer ts.Close()

	boMap := make(map[string]block.BlockOperation)
	kp, blk, boList := prepareOpsWithoutSave(10, storage)

	for _, bo := range boList {
		boMap[bo.Hash] = bo
	}
	ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
	ba.MustSave(storage)

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

			blk.MustSave(storage)
			for _, bo := range boMap {
				bo.MustSave(storage)
			}
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
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
			r.Block = &blk
			txS, err := json.Marshal(r.Resource())
			require.NoError(t, err)
			require.Equal(t, txS, line)

			require.Equal(t, blk.ProposedTime, recv["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, recv["confirmed"].(string))
		}
	}

	wg.Wait()
	storage.Close()
}
