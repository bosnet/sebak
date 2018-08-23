package sebak

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"

	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestGetAccountHandler(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	wg.Add(2)

	// Setting Server
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, GetAccountHandler(storage)).Methods("GET")

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
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n sebakcommon.Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			var cba = &block.BlockAccount{}
			json.Unmarshal(line, cba)
			require.Equal(t, ba.Address, cba.Address)
			require.Equal(t, prev+n, cba.GetBalance())
			prev = cba.GetBalance()
		}
		wg.Done()
	}()

	go func() {
		// Makes Some Events
		for n := 1; n < 20; n++ {
			newBalance, err := ba.GetBalance().Add(sebakcommon.Amount(n))
			require.Nil(t, err)
			ba.Balance = newBalance.String()

			ba.Save(storage)
		}

		wg.Done()
	}()

	wg.Wait()

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

func TestGetAccountTransactionsHandler(t *testing.T) {
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountTransactionsHandlerPattern, GetAccountTransactionsHandler(storage)).Methods("GET")

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
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n sebakcommon.Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bts[n].Serialize()
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}
		wg.Done()
	}()

	// Makes Some Events
	for i := 0; i < 20; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp, err = ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
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

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountOperationsHandlerPattern, GetAccountOperationsHandler(storage)).Methods("GET")

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
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n sebakcommon.Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bos[n].Serialize()
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}
		resp.Body.Close()
		wg.Done()
	}()

	// Makes Some Events
	for i := 0; i < 20; i++ {
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

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp2, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
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

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionByHashHandlerPattern, GetTransactionByHashHandler(storage)).Methods("GET")

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

	// Do stream Request to the Server
	go func() {
		resp, err := ts.Client().Do(req)
		require.Nil(t, err)
		reader := bufio.NewReader(resp.Body)
		line, err := reader.ReadBytes('\n')
		require.Nil(t, err)
		line = bytes.Trim(line, "\n\t ")

		serializedBt, err := bt.Serialize()
		require.Nil(t, err)
		require.Equal(t, serializedBt, line)

		resp.Body.Close()
		wg.Done()
	}()

	bt.Save(storage)

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	var receivedBts BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	require.Equal(t, bt.Hash, receivedBts.Hash, "hash is not same")

}

func TestGetTransactionsHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionsHandlerPattern, GetTransactionsHandler(storage)).Methods("GET")

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
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bts[n].Serialize()
			require.Nil(t, err)
			require.Equal(t, txS, line)
		}

		wg.Done()
	}()

	for i := 0; i < 20; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		require.Nil(t, err)
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	require.Nil(t, err)
	resp2, err := ts.Client().Do(req)
	require.Nil(t, err)
	defer resp2.Body.Close()
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
