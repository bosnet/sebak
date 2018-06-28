package sebak

import (
	"boscoin.io/sebak/lib/storage"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func checkError(t *testing.T, err error){
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountHandler(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	wg.Add(2)

	// Setting Server
	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	checkError(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, GetAccountHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Make Dummy BlockAccount
	ba := testMakeBlockAccount()
	ba.Save(storage)
	prev := ba.GetBalance()

	// Do Request
	url := ts.URL + fmt.Sprintf("/account/%s", ba.Address)
	req, err := http.NewRequest("GET", url, nil)
	checkError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	checkError(t, err)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			checkError(t, err)
			var cba = &BlockAccount{}
			json.Unmarshal(line, cba)
			assert.Equal(t, ba.Address, cba.Address, "not equal")
			assert.Equal(t, prev+n, cba.GetBalance(), "not equal")
			prev = cba.GetBalance()
		}
		resp.Body.Close()
		wg.Done()
	}()

	go func() {
		// Makes Some Events
		for n := 1; n < 20; n++ {
			newBalance, err := ba.GetBalance().Add(Amount(n))
			checkError(t, err)
			ba.Balance = newBalance.String()

			ba.Save(storage)
		}

		wg.Done()
	}()

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	checkError(t, err)
	resp, err = ts.Client().Do(req)
	checkError(t, err)
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	checkError(t, err)
	var cba = &BlockAccount{}
	json.Unmarshal(readByte, cba)
	assert.Equal(t, ba.Address, cba.Address, "not equal")
	assert.Equal(t, ba.GetBalance(), cba.GetBalance(), "not equal")

}

func TestGetAccountTransactionsHandler(t *testing.T) {
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	checkError(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountTransactionsHandlerPattern, GetAccountTransactionsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	checkError(t, err)

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		checkError(t, err)
		bts = append(bts, bt)
	}

	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/transactions", kp.Address())
	req, err := http.NewRequest("GET", url, nil)
	checkError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	checkError(t, err)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			checkError(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bts[n].Serialize()
			checkError(t, err)
			if bytes.Compare(txS, line) != 0 {
				t.Error("not same")
			}
		}
		resp.Body.Close()
		wg.Done()
	}()

	// Makes Some Events
	for i := 0; i < 20; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		checkError(t, err)
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	checkError(t, err)
	resp, err = ts.Client().Do(req)
	checkError(t, err)
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	checkError(t, err)
	var receivedBts []BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	assert.Equal(t, len(bts), len(receivedBts), "length is not same")

	i := 0
	for _, bt := range bts {
		assert.Equal(t, bt.Hash, receivedBts[i].Hash, "hash is not same")
		i++
	}

}

func TestGetAccountOperationsHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	checkError(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountOperationsHandlerPattern, GetAccountOperationsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	checkError(t, err)

	var bos []BlockOperation
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, err = GetBlockOperation(storage, boHash)
			checkError(t, err)
			bos = append(bos, bo)
		}
	}
	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/operations", kp.Address())
	req, err := http.NewRequest("GET", url, nil)
	checkError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	checkError(t, err)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			checkError(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bos[n].Serialize()
			checkError(t, err)
			if bytes.Compare(txS, line) != 0 {
				t.Error("not same")
			}
		}
		resp.Body.Close()
		wg.Done()
	}()

	// Makes Some Events
	for i := 0; i < 20; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, err = GetBlockOperation(storage, boHash)
			checkError(t, err)
			bos = append(bos, bo)
		}
	}

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	checkError(t, err)
	resp, err = ts.Client().Do(req)
	checkError(t, err)
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	checkError(t, err)
	var receivedBos []BlockOperation
	json.Unmarshal(readByte, &receivedBos)

	assert.Equal(t, len(bos), len(receivedBos), "length is not same")

	i := 0
	for _, bo := range bos {
		assert.Equal(t, bo.Hash, receivedBos[i].Hash, "hash is not same")
		i++
	}

}

func TestGetTransactionByHashHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	checkError(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionByHashHandlerPattern, GetTransactionByHashHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, err := keypair.Random()
	checkError(t, err)

	tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
	a, err := tx.Serialize()
	checkError(t, err)
	bt := NewBlockTransactionFromTransaction(tx, a)

	// Do a Request
	url := ts.URL + fmt.Sprintf("/transactions/%s", bt.Hash)
	req, err := http.NewRequest("GET", url, nil)
	checkError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	// Do stream Request to the Server
	go func() {
		resp, err := ts.Client().Do(req)
		checkError(t, err)
		reader := bufio.NewReader(resp.Body)
		line, err := reader.ReadBytes('\n')
		checkError(t, err)
		line = bytes.Trim(line, "\n\t ")

		serializedBt, err := bt.Serialize()
		checkError(t, err)
		if !bytes.Equal(serializedBt, line) {
			assert.Equal(t, serializedBt, line, "not same")
		}

		resp.Body.Close()
		wg.Done()
	}()

	bt.Save(storage)

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	checkError(t, err)
	resp, err := ts.Client().Do(req)
	checkError(t, err)
	reader := bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	checkError(t, err)
	var receivedBts BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	assert.Equal(t, bt.Hash, receivedBts.Hash, "hash is not same")

}

func TestGetTransactionsHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, err := sebakstorage.NewTestMemoryLevelDBBackend()
	checkError(t, err)
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionsHandlerPattern, GetTransactionsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		checkError(t, err)
		bts = append(bts, bt)
	}

	// Do a Request
	url := ts.URL + "/transactions"
	req, err := http.NewRequest("GET", url, nil)
	checkError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := ts.Client().Do(req)
	checkError(t, err)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		for n := 0; n < 10; n++ {
			line, err := reader.ReadBytes('\n')
			checkError(t, err)
			line = bytes.Trim(line, "\n\t ")
			txS, err := bts[n].Serialize()
			checkError(t, err)
			if bytes.Compare(txS, line) != 0 {
				t.Error("not same")
			}
		}

		resp.Body.Close()
		wg.Done()
	}()

	for i := 0; i < 20; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, err := tx.Serialize()
		checkError(t, err)
		bt := NewBlockTransactionFromTransaction(tx, a)
		err = bt.Save(storage)
		checkError(t, err)
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req, err = http.NewRequest("GET", url, nil)
	checkError(t, err)
	resp, err = ts.Client().Do(req)
	checkError(t, err)
	reader = bufio.NewReader(resp.Body)
	readByte, err := ioutil.ReadAll(reader)
	checkError(t, err)
	var receivedBts []BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	assert.Equal(t, len(bts), len(receivedBts), "length is not same")

	i := 0
	for _, bt := range bts {
		assert.Equal(t, bt.Hash, receivedBts[i].Hash, "hash is not same")
		i++
	}

}
