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
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"github.com/stretchr/testify/require"
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
		json.Unmarshal(readByte, &recv)

		require.Equal(t, bt.Hash, recv["hash"], "hash is not the same")
		require.Equal(t, bt.Block, recv["block"], "block is not the same")
	}
}

func TestGetTransactionByHashHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, bt := prepareTxWithoutSave(storage)

	// Wait until request registered to observer
	{
		go func() {
			for {
				observer.BlockTransactionObserver.RLock()
				if len(observer.BlockTransactionObserver.Callbacks) > 0 {
					observer.BlockTransactionObserver.RUnlock()
					break
				}
				observer.BlockTransactionObserver.RUnlock()
			}
			bt.MustSave(storage)
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		respBody := request(ts, GetTransactionsHandlerPattern+"/"+bt.Hash, true)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		line, err := reader.ReadBytes('\n')
		require.NoError(t, err)
		recv := make(map[string]interface{})
		json.Unmarshal(line, &recv)
		require.Equal(t, bt.Hash, recv["hash"], "hash is not the same")
		require.Equal(t, bt.Block, recv["block"], "block is not the same")
	}
	wg.Wait()
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
		json.Unmarshal(readByte, &status)

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
		json.Unmarshal(readByte, &status)

		require.Equal(t, bt.Hash, status.Hash, "hash is not the same")
		require.Equal(t, "confirmed", status.Status, "block is not the same")
	}
}

func TestGetTransactionStatusByHashHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, _, bt := prepareTxWithoutSave(storage)
	tp, err := block.NewTransactionPool(bt.Transaction())
	require.NoError(t, err)

	// Wait until request registered to observer
	{
		go func() {
			for {
				observer.BlockTransactionObserver.RLock()
				if len(observer.BlockTransactionObserver.Callbacks) > 0 {
					observer.BlockTransactionObserver.RUnlock()
					break
				}
				observer.BlockTransactionObserver.RUnlock()
			}
			tp.Save(storage)
			bt.MustSave(storage)
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		respBody := request(ts, strings.Replace(GetTransactionStatusHandlerPattern, "{id}", bt.Hash, -1), true)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	var status resource.TransactionStatus
	{
		line, err := reader.ReadBytes('\n')
		require.NoError(t, err)
		json.Unmarshal(line, &status)
		require.Equal(t, bt.Hash, status.Hash)
		require.Equal(t, "notfound", status.Status)

		line, err = reader.ReadBytes('\n')
		require.NoError(t, err)
		json.Unmarshal(line, &status)
		require.Equal(t, bt.Hash, status.Hash)
		require.Equal(t, "submitted", status.Status)

		line, err = reader.ReadBytes('\n')
		require.NoError(t, err)
		json.Unmarshal(line, &status)
		require.Equal(t, bt.Hash, status.Hash)
		require.Equal(t, "confirmed", status.Status)
	}
	wg.Wait()
}

func TestGetTransactionsHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, btList := prepareTxs(storage, 10)

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
		json.Unmarshal(readByte, &recv)
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

func TestGetTransactionsHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, btList := prepareTxsWithoutSave(10, storage)
	btMap := make(map[string]block.BlockTransaction)
	for _, bt := range btList {
		btMap[bt.Hash] = bt
	}

	// Wait until request registered to observer
	{
		go func() {
			for {
				observer.BlockTransactionObserver.RLock()
				if len(observer.BlockTransactionObserver.Callbacks) > 0 {
					observer.BlockTransactionObserver.RUnlock()
					break
				}
				observer.BlockTransactionObserver.RUnlock()
			}
			for _, bt := range btMap {
				bt.MustSave(storage)
			}
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		respBody := request(ts, GetTransactionsHandlerPattern, true)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		// Discard the first entry (genesis)
		_, err := reader.ReadBytes('\n')
		require.NoError(t, err)
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bt := btMap[recv["hash"].(string)]
			r := resource.NewTransaction(&bt)
			txS, err := json.Marshal(r.Resource())
			require.NoError(t, err)
			require.Equal(t, txS, line)
		}
	}
	wg.Wait()
}

func TestGetTransactionsByAccountHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, btList := prepareTxs(storage, 10)

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
		json.Unmarshal(readByte, &recv)
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

func TestGetTransactionsByAccountHandlerStream(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	btMap := make(map[string]block.BlockTransaction)
	kp, btList := prepareTxsWithoutSave(10, storage)
	for _, bt := range btList {
		btMap[bt.Hash] = bt
	}

	// Wait until request registered to observer
	{
		go func() {
			for {
				observer.BlockTransactionObserver.RLock()
				if len(observer.BlockTransactionObserver.Callbacks) > 0 {
					observer.BlockTransactionObserver.RUnlock()
					break
				}
				observer.BlockTransactionObserver.RUnlock()
			}
			for _, bt := range btMap {
				bt.MustSave(storage)
			}
			wg.Done()
		}()
	}

	// Do a Request
	var reader *bufio.Reader
	{
		url := strings.Replace(GetAccountTransactionsHandlerPattern, "{id}", kp.Address(), -1)
		respBody := request(ts, url, true)
		defer respBody.Close()
		reader = bufio.NewReader(respBody)
	}

	// Check the output
	{
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.NoError(t, err)
			line = bytes.Trim(line, "\n\t ")
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			bt := btMap[recv["hash"].(string)]
			r := resource.NewTransaction(&bt)
			txS, err := json.Marshal(r.Resource())
			require.NoError(t, err)
			require.Equal(t, txS, line)
		}
	}
	wg.Wait()
}

func TestGetTransactionsHandlerPage(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	_, btList := prepareTxs(storage, 10)

	requestFunction := func(url string) ([]interface{}, map[string]interface{}) {
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

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
