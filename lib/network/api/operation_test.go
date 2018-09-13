package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"github.com/stretchr/testify/require"
	"strings"
	"sync"
)

func TestGetOperationsByAccountHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, boList, err := prepareOps(storage, 0, 10, nil)
	require.Nil(t, err)

	// Do a Request
	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	respBody, err := request(ts, url, false)
	require.Nil(t, err)
	defer respBody.Close()
	reader := bufio.NewReader(respBody)

	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)

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

func TestGetOperationsByAccountHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	boMap := make(map[string]block.BlockOperation)
	kp, boList, err := prepareOpsWithoutSave(0, 10, nil)
	require.Nil(t, err)
	for _, bo := range boList {
		boMap[bo.Hash] = bo
	}

	// Wait until request registered to observer
	{
		var notify = make(chan struct{})
		go func() {
			<-notify
			for _, bo := range boMap {
				bo.Save(storage)
			}
			wg.Done()
		}()

		go func() {
			for _, ok := observer.BlockOperationObserver.Callbacks["saved"]; !ok; {
				break
			}
			close(notify)
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
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
