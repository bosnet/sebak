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

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, GetAccountHandler(storage)).Methods("GET")

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
	require.Equal(t, resp.StatusCode, 404)
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

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		bts = append(bts, bt)
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
			bt := NewBlockTransactionFromTransaction(tx, a)
			err = bt.Save(storage)
			if !assert.Nil(t, err) {
				panic(err)
			}
			bts = append(bts, bt)
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
		txS, err := bts[n].Serialize()
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

	require.Equal(t, len(bts), len(receivedBts), "length is not same")

	i := 0
	for _, bt := range bts {
		require.Equal(t, bt.Hash, receivedBts[i].Hash, "hash is not same")
		i++
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

	var bos []BlockOperation
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, err = GetBlockOperation(storage, boHash)
			require.Nil(t, err)
			bos = append(bos, bo)
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
			bt := NewBlockTransactionFromTransaction(tx, a)
			bt.Save(storage)

			for _, boHash := range bt.Operations {
				var bo BlockOperation
				bo, err = GetBlockOperation(storage, boHash)
				if !assert.Nil(t, err) {
					panic(err)
				}
				bos = append(bos, bo)
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
		txS, err := bos[n].Serialize()
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

	require.Equal(t, len(bos), len(receivedBos), "length is not same")

	i := 0
	for _, bo := range bos {
		require.Equal(t, bo.Hash, receivedBos[i].Hash, "hash is not same")
		i++
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
	bt := NewBlockTransactionFromTransaction(tx, a)

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

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		bts = append(bts, bt)
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
			bt := NewBlockTransactionFromTransaction(tx, a)
			err = bt.Save(storage)
			if !assert.Nil(t, err) {
				panic(err)
			}
			bts = append(bts, bt)
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
		txS, err := bts[n].Serialize()
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

	require.Equal(t, len(bts), len(receivedBts), "length is not same")

	i := 0
	for _, bt := range bts {
		require.Equal(t, bt.Hash, receivedBts[i].Hash, "hash is not same")
		i++
	}
}
