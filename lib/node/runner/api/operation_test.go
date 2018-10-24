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

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stretchr/testify/require"
)

func TestGetOperationsByAccountHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.NoError(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, boList, err := prepareOps(storage, 10, nil)
	require.NoError(t, err)

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
		respBody, err := request(ts, url, false)
		require.NoError(t, err)
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
		}
	}
}

func TestGetOperationsByAccountHandlerWithType(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.NoError(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, boList, err := prepareOps(storage, 10, nil)
	require.NoError(t, err)
	ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
	ba.MustSave(storage)

	// Do a Request
	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	{
		url := url + "?type=" + string(operation.TypeCreateAccount)
		respBody, err := request(ts, url, false)
		require.NoError(t, err)
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
		respBody, err := request(ts, url, false)
		require.NoError(t, err)
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
		}
	}

}

func TestGetOperationsByAccountHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage, err := prepareAPIServer()
	require.NoError(t, err)
	defer storage.Close()
	defer ts.Close()

	boMap := make(map[string]block.BlockOperation)
	kp, boList, err := prepareOpsWithoutSave(10, nil)
	require.NoError(t, err)
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
		respBody, err := request(ts, url, true)
		require.NoError(t, err)
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
