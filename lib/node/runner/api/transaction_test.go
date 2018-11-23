package api

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node/runner/api/resource"
)

func TestGetTransactionByHashHandler(t *testing.T) {

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, bt := prepareTxWithoutSave(storage)
	bt.MustSave(storage)

	{ // unknown transaction
		req, _ := http.NewRequest("GET", ts.URL+GetTransactionsHandlerPattern+"/findme", nil)
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}

	var reader *bufio.Reader
	// Do a Request
	{
		respBody := request(ts, GetTransactionsHandlerPattern+"/"+bt.Hash, false)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}
	// Check the output
	{
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)

		require.Equal(t, bt.Hash, recv["hash"], "hash is not the same")
		require.Equal(t, bt.Block, recv["block"], "block is not the same")
	}
}

func TestGetTransactionStatusByHashHandler(t *testing.T) {

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, bt := prepareTxWithoutSave(storage)
	bt.MustSave(storage)

	var reader *bufio.Reader
	{ // unknown transaction
		respBody := request(ts, strings.Replace(GetTransactionStatusHandlerPattern, "{id}", "findme", -1), false)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		var status resource.TransactionStatus
		common.MustUnmarshalJSON(readByte, &status)

		require.Equal(t, status.Hash, "findme")
		require.Equal(t, status.Status, "notfound")
	}

	// Do a Request
	{
		respBody := request(ts, strings.Replace(GetTransactionStatusHandlerPattern, "{id}", bt.Hash, -1), false)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}
	// Check the output
	{
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		var status resource.TransactionStatus
		common.MustUnmarshalJSON(readByte, &status)

		require.Equal(t, bt.Hash, status.Hash, "hash is not the same")
		require.Equal(t, "confirmed", status.Status, "block is not the same")
	}
}

func TestGetTransactionsHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, btList := prepareTxs(storage, 10)

	var reader *bufio.Reader
	{
		// Do a Request
		respBody := request(ts, GetTransactionsHandlerPattern, false)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(btList)+1, len(records), "length is not the same")

		for i, r := range records[1:] {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)
			block := bt["block"].(string)

			require.Equal(t, hash, btList[i].Hash, "hash is not the same")
			require.Equal(t, block, btList[i].Block, "block is not the same")
		}
	}
}

func TestGetTransactionsByAccountHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, _, btList := prepareTxs(storage, 10)

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetAccountTransactionsHandlerPattern, "{id}", kp.Address(), -1)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(btList), len(records), "length is not the same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)
			block := bt["block"].(string)

			require.Equal(t, hash, btList[i].Hash, "hash is not the same")
			require.Equal(t, block, btList[i].Block, "block is not the same")
		}
	}
}

func TestGetTransactionsHandlerPage(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, btList := prepareTxs(storage, 10)

	requestFunction := func(url string) ([]interface{}, map[string]interface{}) {
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
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
		require.Equal(t, len(btList), len(records[1:]), "length is not the same")

		for i, r := range records[1:] {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[i].Hash, "hash is not the same")
			require.Equal(t, bt["block"], btList[i].Block, "block is not the same")
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "6", 1)
		query = strings.Replace(query, "{reverse}", "false", 1)
		records, links := testFunction(query)
		require.Equal(t, len(btList[:5]), len(records[1:]), "length is not the same")

		for i, r := range records[1:] {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[i].Hash, "hash is not the same")
			require.Equal(t, bt["block"], btList[i].Block, "block is not the same")
		}

		nextLink := links["next"].(map[string]interface{})["href"].(string)

		{
			records, _ := requestFunction(nextLink)
			require.Equal(t, len(btList[5:]), len(records), "length is not the same")

			for i, r := range records {
				bt := r.(map[string]interface{})
				require.Equal(t, bt["hash"], btList[5+i].Hash, "hash is not the same")
				require.Equal(t, bt["block"], btList[5+i].Block, "block is not the same")
			}
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "0", 1)
		query = strings.Replace(query, "{reverse}", "true", 1)
		records, _ := testFunction(query)
		require.Equal(t, len(btList), len(records[:len(records)-1]), "length is not the same")

		for i, r := range records[:len(records)-1] {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[len(btList)-1-i].Hash, "hash is not the same")
			require.Equal(t, bt["block"], btList[len(btList)-1-i].Block, "block is not the same")
		}
	}
	{
		query := strings.Replace(QueryPattern, "{cursor}", "", 1)
		query = strings.Replace(query, "{limit}", "5", 1)
		query = strings.Replace(query, "{reverse}", "true", 1)
		records, _ := testFunction(query)
		require.Equal(t, len(btList[5:]), len(records), "length is not the same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, bt["hash"], btList[len(btList)-1-i].Hash, "hash is not the same")
			require.Equal(t, bt["block"], btList[len(btList)-1-i].Block, "block is not the same")

		}
	}
}
