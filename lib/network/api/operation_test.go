package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network/api/resource"
	"github.com/stretchr/testify/require"
	"strings"
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
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, boList, err := prepareOps(storage, 0, 10, nil)
	require.Nil(t, err)
	{
		// Do a Request
		url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
		respBody, err := request(ts, url, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		// Producer
		recv := make(chan struct{})
		go func() {
			_, boList2, err := prepareOps(storage, 1, 10, kp)
			require.Nil(t, err)
			boList = append(boList, boList2...)
			close(recv)
		}()
		var boMap = make(map[string]block.BlockOperation)
		for _, bo := range boList {
			boMap[bo.Hash] = bo
		}

		// Do stream Request to the Server
		<-recv
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
}
