package api

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

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAccountHandler(t *testing.T) {
	// Setting Server
	storage, err := storage.NewTestMemoryLevelDBBackend()
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
			ba.Balance = ba.GetBalance().MustAdd(common.Amount(n))
			ba.Save(storage)
			if n <= 10 {
				recv <- struct{}{}
			}
		}
		close(recv)
	}()

	// Do stream Request to the Server
	var n common.Amount
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
	//var cba = &block.BlockAccount{}
	cba := map[string]interface{}{}
	json.Unmarshal(readByte, cba)
	require.Equal(t, ba.Address, cba["Address"], "not equal")
	require.Equal(t, ba.GetBalance(), cba["Balance"], "not equal")
}

// Test that getting an inexisting account returns an error
func TestGetNonExistentAccountHandler(t *testing.T) {
	// Setting Server
	storage, err := storage.NewTestMemoryLevelDBBackend()
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

	storage, err := storage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountTransactionsHandlerPattern, apiHandler.GetAccountTransactionsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	var txs []transaction.Transaction
	var txHashes []string
	var btmap = make(map[string]block.BlockTransaction)
	for i := 0; i < 5; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
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
			tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)

			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
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
	var n common.Amount
	for n = 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")
		var receivedBt block.BlockTransaction
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
	var receivedBts []block.BlockTransaction
	json.Unmarshal(readByte, &receivedBts)
	fmt.Println(receivedBts[0])

	require.Equal(t, len(btmap), len(receivedBts), "length is not same")

	for _, bt := range receivedBts {
		require.Equal(t, bt.Hash, btmap[bt.Hash].Hash, "hash is not same")
	}
}

func TestGetAccountOperationsHandler(t *testing.T) {
	storage, err := storage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountOperationsHandlerPattern, apiHandler.GetAccountOperationsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	var txs []transaction.Transaction
	var txHashes []string
	var bomap = make(map[string]block.BlockOperation)
	for i := 0; i < 5; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 3, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo block.BlockOperation
			bo, err = block.GetBlockOperation(storage, boHash)
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
			tx := transaction.TestMakeTransactionWithKeypair(networkID, 3, kp)
			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
			bt.Save(storage)

			for _, boHash := range bt.Operations {
				var bo block.BlockOperation
				bo, err = block.GetBlockOperation(storage, boHash)
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
	var n common.Amount
	for n = 0; n < 10; n++ {
		<-recv
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")
		var receivedBo block.BlockOperation
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
	var receivedBos []block.BlockOperation
	json.Unmarshal(readByte, &receivedBos)

	require.Equal(t, len(bomap), len(receivedBos), "length is not same")

	for _, bo := range receivedBos {
		require.Equal(t, bo.Hash, bomap[bo.Hash].Hash, "hash is not same")
	}
}

func TestGetTransactionByHashHandler(t *testing.T) {
	storage, err := storage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionByHashHandlerPattern, apiHandler.GetTransactionByHashHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	require.Nil(t, err)

	tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
	a, err := tx.Serialize()
	require.Nil(t, err)

	theBlock := block.TestMakeNewBlock([]string{tx.GetHash()})
	bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)

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
	var receivedBts block.BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, bt.Hash, receivedBts.Hash, "hash is not same")
}

func TestGetTransactionsHandler(t *testing.T) {
	storage, err := storage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionsHandlerPattern, apiHandler.GetTransactionsHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	var txs []transaction.Transaction
	var txHashes []string
	var btmap = make(map[string]block.BlockTransaction)
	for i := 0; i < 5; i++ {
		_, tx := transaction.TestMakeTransaction(networkID, 1)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
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
			_, tx := transaction.TestMakeTransaction(networkID, 1)

			a, err := tx.Serialize()
			if !assert.Nil(t, err) {
				panic(err)
			}
			bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
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
		var receivedBt block.BlockTransaction
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
	var receivedBts []block.BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, len(receivedBts), len(receivedBts), "length is not same")

	for _, bt := range receivedBts {
		require.Equal(t, bt.Hash, btmap[bt.Hash].Hash, "hash is not same")
	}
}

func TestProblem(t *testing.T) {

	router := mux.NewRouter()

	statusProblem := errors.NewStatusProblem(http.StatusBadRequest)
	detailedStatusProblem := errors.NewDetailedStatusProblem(http.StatusBadRequest, "paramaters are not enough")
	errorProblem := errors.NewErrorProblem(errors.ErrorInvalidOperation)

	router.HandleFunc("/problem_status_default", func(w http.ResponseWriter, r *http.Request) {
		statusProblem.Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_with_detail", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_with_detail_instance", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.SetInstance("http://boscoin.io/httperror/details/1").Problem(w, "", -1)
	})

	router.HandleFunc("/problem_status_default_with_detail", func(w http.ResponseWriter, r *http.Request) {
		detailedStatusProblem.Problem(w, "bad request yo!", -1)
	})

	router.HandleFunc("/problem_with_error", func(w http.ResponseWriter, r *http.Request) {
		errorProblem.Problem(w, "", -1)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	// problem_status_default
	{
		url := ts.URL + fmt.Sprintf("/problem_status_default")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		//fmt.Printf("%s\n", readByte)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := statusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Empty(t, m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_status_with_detail
	{
		url := ts.URL + fmt.Sprintf("/problem_status_with_detail")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		//fmt.Printf("%s\n", readByte)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, p.Detail, m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_status_with_detail_instance
	{
		url := ts.URL + fmt.Sprintf("/problem_status_with_detail_instance")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		//fmt.Printf("%s\n", readByte)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem.SetInstance("http://boscoin.io/httperror/details/1")
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, p.Detail, m["detail"])
			require.Equal(t, p.Instance, m["instance"])
		}
	}

	// problem_status_default_with_detail
	{
		url := ts.URL + fmt.Sprintf("/problem_status_default_with_detail")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		//fmt.Printf("%s\n", readByte)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := detailedStatusProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Equal(t, float64(p.Status), m["status"])
			require.Equal(t, "bad request yo!", m["detail"])
			require.Empty(t, m["instance"])
		}
	}

	// problem_with_error
	{
		url := ts.URL + fmt.Sprintf("/problem_with_error")
		resp, err := http.Get(url)
		require.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		//fmt.Printf("%s\n", readByte)
		{
			var f interface{}
			json.Unmarshal(readByte, &f)
			m := f.(map[string]interface{})
			p := errorProblem
			require.Equal(t, p.Type, m["type"])
			require.Equal(t, p.Title, m["title"])
			require.Empty(t, m["status"])
			require.Empty(t, m["detail"])
			require.Empty(t, m["instance"])
		}
	}
}
