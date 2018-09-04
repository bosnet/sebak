package sebak

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"

	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
)

func TestGetAccountHandler(t *testing.T) {
	// Setting Server
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Make Dummy BlockAccount
	ba := block.TestMakeBlockAccount()
	ba.Save(storage)
	prev := ba.GetBalance()

	// Do Request
	url := ts.URL + fmt.Sprintf("/account/%s", ba.Address)
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader := bufio.NewReader(resp.Body)

	recv := make(chan struct{})
	go func() {
		// Makes Some Events
		for n := 1; n < 20; n++ {
			newBalance := ba.GetBalance().MustAdd(sebakcommon.Amount(n))
			ba.Balance = newBalance.String()
			ba.Save(storage)
			if n <= 10 {
				recv <- struct{}{}
			}
		}
		close(recv)
	}()

	// Do stream Request to the Server
	var n sebakcommon.Amount
	for n = 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		var cba = &block.BlockAccount{}
		json.Unmarshal(line, cba)
		require.Equal(t, ba.Address, cba.Address)
		require.Equal(t, prev+n, cba.GetBalance())
		prev = cba.GetBalance()
	}
	<-recv // Close

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp, err = ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var cba = &block.BlockAccount{}
	json.Unmarshal(readByte, cba)
	require.Equal(t, ba.Address, cba.Address, "not equal")
	require.Equal(t, ba.GetBalance(), cba.GetBalance(), "not equal")
}

// Test that getting an inexisting account returns an error
func TestGetNonExistentAccountHandler(t *testing.T) {
	// Setting Server
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Do the request to an inexisting address
	genesisAddress := "GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP"
	url := ts.URL + fmt.Sprintf("/account/%s", genesisAddress)
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, 404, resp.StatusCode)
	reader := bufio.NewReader(resp.Body)
	data, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, "Not Found\n", string(data))
}

func TestGetAccountTransactionsHandler(t *testing.T) {
	var err error

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountTransactionsHandlerPattern, apiHandler.GetAccountTransactionsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	var txs []Transaction
	var txHashes []string
	var btmap = make(map[string]BlockTransaction)
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		btmap[bt.Hash] = bt
	}

	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/transactions", kp.Address())
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader := bufio.NewReader(resp.Body)

	// Makes Some Events
	recv := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
			err = bt.Save(storage)
			if !assert.Nil(t, err) {
				panic(err)
			}
			btmap[bt.Hash] = bt
			if i < 10 {
				recv <- struct{}{}
			}
		}
		close(recv)
	}()

	// Do stream Request to the Server
	var n sebakcommon.Amount
	for n = 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")
		var receivedBt BlockTransaction
		json.Unmarshal(line, &receivedBt)
		txS, err := btmap[receivedBt.Hash].Serialize()
		require.Nil(t, err)
		require.Equal(t, txS, line)
	}
	<-recv // Wait for close

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp, err = ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var receivedBts []BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, len(btmap), len(receivedBts), "length is not same")

	for _, bt := range receivedBts {
		require.Equal(t, bt.Hash, btmap[bt.Hash].Hash, "hash is not same")
	}
}

func TestGetAccountOperationsHandler(t *testing.T) {
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountOperationsHandlerPattern, apiHandler.GetAccountOperationsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	var txs []Transaction
	var txHashes []string
	var bomap = make(map[string]BlockOperation)
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, err = GetBlockOperation(storage, boHash)
			require.Nil(t, err)
			bomap[bo.Hash] = bo
		}
	}
	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/operations", kp.Address())
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader := bufio.NewReader(resp.Body)

	// Makes Some Events
	recv := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
			bt.Save(storage)

			for _, boHash := range bt.Operations {
				var bo BlockOperation
				bo, err = GetBlockOperation(storage, boHash)
				if !assert.Nil(t, err) {
					panic(err)
				}
				bomap[bo.Hash] = bo
			}
			if i < 10 {
				recv <- struct{}{}
			}
		}
		close(recv)
	}()

	// Do stream Request to the Server
	var n sebakcommon.Amount
	for n = 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")
		var receivedBo BlockOperation
		json.Unmarshal(line, &receivedBo)
		txS, err := bomap[receivedBo.Hash].Serialize()
		require.Nil(t, err)
		require.Equal(t, txS, line)
	}
	<-recv // Wait for close

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp2, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, 200)
	reader = bufio.NewReader(resp2.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var receivedBos []BlockOperation
	json.Unmarshal(readByte, &receivedBos)

	require.Equal(t, len(bomap), len(receivedBos), "length is not same")

	for _, bo := range receivedBos {
		require.Equal(t, bo.Hash, bomap[bo.Hash].Hash, "hash is not same")
	}
}

func TestGetTransactionByHashHandler(t *testing.T) {
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionByHashHandlerPattern, apiHandler.GetTransactionByHashHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
	a, err := tx.Serialize()
	require.Nil(t, err)

	block := testMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)

	// Do a Request
	url := ts.URL + fmt.Sprintf("/transactions/%s", bt.Hash)
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")

	// Produce an event
	bt.Save(storage)

	// Do stream Request to the Server
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadBytes('\n')
	require.Nil(t, err)
	line = bytes.Trim(line, "\n\t ")

	serializedBt, err := bt.Serialize()
	require.Nil(t, err)
	require.Equal(t, serializedBt, line)

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp2, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, 200)
	reader = bufio.NewReader(resp2.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var receivedBts BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, bt.Hash, receivedBts.Hash, "hash is not same")
}

func TestGetTransactionsHandler(t *testing.T) {
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionsHandlerPattern, apiHandler.GetTransactionsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	var txs []Transaction
	var txHashes []string
	var btmap = make(map[string]BlockTransaction)
	for i := 0; i < 5; i++ {
		_, tx := TestMakeTransaction(networkID, 1)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		btmap[bt.Hash] = bt
	}

	// Do a Request
	url := ts.URL + "/transactions"
	req, err := http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, 200)
	reader := bufio.NewReader(resp.Body)

	// Producer
	recv := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			_, tx := TestMakeTransaction(networkID, 1)

			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
			err = bt.Save(storage)
			if !assert.Nil(t, err) {
				panic(err)
			}
			btmap[bt.Hash] = bt
			if i < 10 {
				recv <- struct{}{}
			}
		}
		close(recv)
	}()

	// Do stream Request to the Server
	for n := 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")
		var receivedBt BlockTransaction
		json.Unmarshal(line, &receivedBt)
		txS, err := btmap[receivedBt.Hash].Serialize()
		require.Nil(t, err)
		require.Equal(t, txS, line)
	}
	<-recv // close

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp2, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, 200)
	reader = bufio.NewReader(resp2.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var receivedBts []BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, len(receivedBts), len(receivedBts), "length is not same")

	for _, bt := range receivedBts {
		require.Equal(t, bt.Hash, btmap[bt.Hash].Hash, "hash is not same")
	}
}

func TestAPIResourceAccount(t *testing.T) {
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	// Account
	{
		ba := block.TestMakeBlockAccount()
		ba.Save(storage)
		ra := &APIResourceAccount{
			accountId:  ba.Address,
			checkpoint: ba.Checkpoint,
			balance:    ba.Balance,
		}
		r := ra.Resource(ra.LinkSelf())
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, ba.Address, m["account_id"])
			require.Equal(t, ba.Address, m["id"])
			require.Equal(t, ba.Checkpoint, m["checkpoint"])
			require.Equal(t, ba.Balance, m["balance"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlAccounts, "{id}", ba.Address, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// Transaction
	{
		_, tx := TestMakeTransaction(networkID, 1)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		rt := &APIResourceTransaction{
			hash:               bt.Hash,
			previousCheckpoint: bt.PreviousCheckpoint,
			sourceCheckpoint:   bt.SourceCheckpoint,
			targetCheckpoint:   bt.TargetCheckpoint,
			signature:          bt.Signature,
			source:             bt.Source,
			fee:                bt.Fee.String(),
			amount:             bt.Amount.String(),
			created:            bt.Created,
			operations:         bt.Operations,
		}
		r := rt.Resource(rt.LinkSelf())
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bt.Hash, m["id"])
			require.Equal(t, bt.Hash, m["hash"])
			require.Equal(t, bt.Source, m["account"])
			require.Equal(t, bt.Fee.String(), m["fee_paid"])
			require.Equal(t, bt.SourceCheckpoint, m["source_checkpoint"])
			require.Equal(t, bt.TargetCheckpoint, m["target_checkpoint"])
			require.Equal(t, bt.Created, m["created_at"])
			require.Equal(t, float64(len(bt.Operations)), m["operation_count"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlTransactions, "{id}", bt.Hash, -1), l["self"].(map[string]interface{})["href"])
		}

	}

	// Operation
	{
		_, tx := TestMakeTransaction(networkID, 1)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)
		bo, err := GetBlockOperation(storage, bt.Operations[0])

		ro := &APIResourceOperation{
			hash:    bo.Hash,
			txHash:  bo.TxHash,
			funder:  bo.Source,
			account: bo.Target,
			otype:   string(bo.Type),
			amount:  bo.Amount.String(),
		}
		r := ro.Resource(ro.LinkSelf())
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bo.Hash, m["id"])
			require.Equal(t, bo.Hash, m["hash"])
			require.Equal(t, bo.Source, m["funder"])
			require.Equal(t, bo.Target, m["account"])
			require.Equal(t, string(bo.Type), m["type"])
			require.Equal(t, bo.Amount.String(), m["amount"])
			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// List
	{
		_, tx := TestMakeTransaction(networkID, 3)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		rol := APIResourceList{}
		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, err = GetBlockOperation(storage, boHash)
			require.Nil(t, err)

			ro := &APIResourceOperation{
				hash:    bo.Hash,
				txHash:  bo.TxHash,
				funder:  bo.Source,
				account: bo.Target,
				otype:   string(bo.Type),
				amount:  bo.Amount.String(),
			}
			rol = append(rol, ro)
		}

		urlneedToBeFilledByAPI := "/operations/"
		r := rol.Resource(urlneedToBeFilledByAPI)
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{

			var f interface{}

			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})

			l := m["_links"].(map[string]interface{})
			require.Equal(t, urlneedToBeFilledByAPI, l["self"].(map[string]interface{})["href"])

			records := m["_embedded"].(map[string]interface{})["records"].([]interface{})
			for _, v := range records {
				record := v.(map[string]interface{})
				id := record["id"].(string)
				bo, err := GetBlockOperation(storage, id)
				require.Nil(t, err)
				require.Equal(t, bo.Hash, record["id"])
				require.Equal(t, bo.Hash, record["hash"])
				require.Equal(t, bo.Source, record["funder"])
				require.Equal(t, bo.Target, record["account"])
				require.Equal(t, string(bo.Type), record["type"])
				require.Equal(t, bo.Amount.String(), record["amount"])
				l := record["_links"].(map[string]interface{})
				require.Equal(t, strings.Replace(UrlOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
			}
		}
	}
}
