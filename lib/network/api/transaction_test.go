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

func TestGetTransactionByHashHandler(t *testing.T) {

	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, _, bt, err := prepareTx()
	require.Nil(t, err)
	bt.Save(storage)
	{
		// Do a Request
		respBody, err := request(ts, GetTransactionsHandlerPattern+"/"+bt.Hash, false)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)

		require.Equal(t, bt.Hash, recv["hash"], "hash is not same")
	}
}

func TestGetTransactionByHashHandlerStream(t *testing.T) {

	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, _, bt, err := prepareTx()
	require.Nil(t, err)
	{
		// Do a Request
		respBody, err := request(ts, GetTransactionsHandlerPattern+"/"+bt.Hash, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		{
			err = bt.Save(storage)
			require.Nil(t, err)
		}

		for {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n")
			if line == nil {
				continue
			}
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			require.Equal(t, bt.Hash, recv["hash"], "hash is not same")
			break
		}
	}
}

func TestGetTransactionsHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, btList, err := prepareTxs(storage, 0, 10, nil)
	require.Nil(t, err)

	{
		// Do a Request
		respBody, err := request(ts, GetTransactionsHandlerPattern, false)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)

		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(btList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, btList[i].Hash, "hash is not same")
		}
	}
}

func TestGetTransactionsHandlerStream(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, btList, err := prepareTxs(storage, 0, 10, nil)
	require.Nil(t, err)

	// streaming
	{
		// Do a Request
		respBody, err := request(ts, GetTransactionsHandlerPattern, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		// Producer
		{
			_, btList2, err := prepareTxs(storage, 1, 10, nil)
			require.Nil(t, err)
			btList = append(btList, btList2...)
		}
		var btMap = make(map[string]block.BlockTransaction)
		for _, bt := range btList {
			btMap[bt.Hash] = bt
		}

		// Do stream Request to the Server
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bt := btMap[recv["hash"].(string)]
			r := resource.NewTransaction(&bt)
			txS, err := json.Marshal(r.Resource())
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}
	}
}

func TestGetTransactionsByAccountHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, btList, err := prepareTxs(storage, 0, 10, nil)
	require.Nil(t, err)

	// Do a Request
	url := strings.Replace(GetAccountTransactionsHandlerPattern, "{id}", kp.Address(), -1)
	respBody, err := request(ts, url, false)
	require.Nil(t, err)
	defer respBody.Close()
	reader := bufio.NewReader(respBody)

	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)

	recv := make(map[string]interface{})
	json.Unmarshal(readByte, &recv)
	records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

	require.Equal(t, len(btList), len(records), "length is not same")

	for i, r := range records {
		bt := r.(map[string]interface{})
		hash := bt["hash"].(string)

		require.Equal(t, hash, btList[i].Hash, "hash is not same")
	}

}

func TestGetTransactionsByAccountHandlerStream(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	kp, btList, err := prepareTxs(storage, 0, 10, nil)
	require.Nil(t, err)
	{
		// Do a Request
		url := strings.Replace(GetAccountTransactionsHandlerPattern, "{id}", kp.Address(), -1)
		respBody, err := request(ts, url, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		// Producer
		{
			_, btList2, err := prepareTxs(storage, 1, 10, kp)
			require.Nil(t, err)
			btList = append(btList, btList2...)
		}
		var btMap = make(map[string]block.BlockTransaction)
		for _, bt := range btList {
			btMap[bt.Hash] = bt
		}

		// Do stream Request to the Server
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bt := btMap[recv["hash"].(string)]
			r := resource.NewTransaction(&bt)
			txS, err := json.Marshal(r.Resource())
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}
	}
}

func TestGetTransactionsHandlerPage(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	_, btList, err := prepareTxs(storage, 0, 10, nil)
	require.Nil(t, err)

	requestFunction := func(url string) ([]interface{}, map[string]interface{}) {
		respBody, err := request(ts, url, false)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)

		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		links := recv["_links"].(map[string]interface{})
		return records, links
	}

	testFunction := func(query string) ([]interface{}, map[string]interface{}) {
		return requestFunction(GetTransactionsHandlerPattern + "?" + query)
	}

	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "0", 1)
		query = strings.Replace(query, "{reverse}", "false", 1)
		records, _ := testFunction(query)
		require.Equal(t, len(btList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[i].Hash, "hash is not same")
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "5", 1)
		query = strings.Replace(query, "{reverse}", "false", 1)
		records, links := testFunction(query)
		require.Equal(t, len(btList[:5]), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[i].Hash, "hash is not same")
		}

		nextLink := links["next"].(map[string]interface{})["href"].(string)

		{
			records, _ := requestFunction(nextLink)
			require.Equal(t, len(btList[5:]), len(records), "length is not same")

			for i, r := range records {
				bt := r.(map[string]interface{})
				require.Equal(t, bt["hash"], btList[5+i].Hash, "hash is not same")
			}
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "0", 1)
		query = strings.Replace(query, "{reverse}", "true", 1)
		records, _ := testFunction(query)
		require.Equal(t, len(btList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[len(btList)-1-i].Hash, "hash is not same")
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "5", 1)
		query = strings.Replace(query, "{reverse}", "true", 1)
		records, _ := testFunction(query)
		require.Equal(t, len(btList[5:]), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[len(btList)-1-i].Hash, "hash is not same")
		}
	}

	//TODO: cursor
}
