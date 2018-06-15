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

func prepareServer(pattern string, getHandlerFunc func(storage *sebakstorage.LevelDBBackend) http.HandlerFunc) (testServer *httptest.Server, storage *sebakstorage.LevelDBBackend, err error) {
	// Setting Server
	storage, err = sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		return
	}

	router := mux.NewRouter()
	router.HandleFunc(pattern, getHandlerFunc(storage)).Methods("GET")

	testServer = httptest.NewServer(router)
	return
}

func TestGetAccountHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(2)

	// Setting Server
	storage, _ := sebakstorage.NewTestMemoryLevelDBBackend()
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
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, _ := http.DefaultClient.Do(req)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, _ := reader.ReadBytes('\n')
			var cba = &BlockAccount{}
			json.Unmarshal(line, cba)
			//fmt.Println(string(line))
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
			newBalance, _ := ba.GetBalance().Add(Amount(n))
			ba.Balance = newBalance.String()

			ba.Save(storage)
		}

		wg.Done()
	}()

	wg.Wait()

	// No streaming
	req.Header.Del("Accept")
	resp, _ = http.DefaultClient.Do(req)
	reader = bufio.NewReader(resp.Body)
	readByte, _ := ioutil.ReadAll(reader)
	var cba = &BlockAccount{}
	json.Unmarshal(readByte, cba)
	assert.Equal(t, ba.Address, cba.Address, "not equal")
	assert.Equal(t, ba.GetBalance(), cba.GetBalance(), "not equal")

}

func TestGetAccountTransactionsHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountTransactionsHandlerPattern, GetAccountTransactionsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, _ := keypair.Random()

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(storage)
		if err != nil {
			t.Error(err)
			return
		}
		bts = append(bts, bt)
	}

	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/transactions", kp.Address())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, _ := http.DefaultClient.Do(req)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, _ := reader.ReadBytes('\n')
			line = bytes.Trim(line, "\n\t ")
			txS, _ := bts[n].Serialize()
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

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(storage)
		if err != nil {
			t.Error(err)
			return
		}
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req.Header.Del("Accept")
	resp, _ = http.DefaultClient.Do(req)
	reader = bufio.NewReader(resp.Body)
	readByte, _ := ioutil.ReadAll(reader)
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

	storage, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetAccountOperationsHandlerPattern, GetAccountOperationsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, _ := keypair.Random()

	var bos []BlockOperation
	for i := 0; i < 5; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 3, kp)
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, _ = GetBlockOperation(storage, boHash)
			bos = append(bos, bo)
		}
	}
	// Do a Request
	url := ts.URL + fmt.Sprintf("/account/%s/operations", kp.Address())
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, _ := http.DefaultClient.Do(req)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		var n Amount
		for n = 0; n < 10; n++ {
			line, _ := reader.ReadBytes('\n')
			line = bytes.Trim(line, "\n\t ")
			txS, _ := bos[n].Serialize()
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
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		bt.Save(storage)

		for _, boHash := range bt.Operations {
			var bo BlockOperation
			bo, _ = GetBlockOperation(storage, boHash)
			bos = append(bos, bo)
		}
	}

	wg.Wait()

	// No streaming
	req.Header.Del("Accept")
	resp, _ = http.DefaultClient.Do(req)
	reader = bufio.NewReader(resp.Body)
	readByte, _ := ioutil.ReadAll(reader)
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

	storage, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionByHashHandlerPattern, GetTransactionByHashHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	kp, _ := keypair.Random()

	tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
	a, _ := tx.Serialize()
	bt := NewBlockTransactionFromTransaction(tx, a)

	// Do a Request
	url := ts.URL + fmt.Sprintf("/transactions/%s", bt.Hash)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")

	// Do stream Request to the Server
	go func() {
		resp, _ := http.DefaultClient.Do(req)
		reader := bufio.NewReader(resp.Body)
		line, _ := reader.ReadBytes('\n')
		line = bytes.Trim(line, "\n\t ")

		serializedBt, _ := bt.Serialize()
		if !bytes.Equal(serializedBt, line) {
			assert.Equal(t, serializedBt, line, "not same")
		}

		resp.Body.Close()
		wg.Done()
	}()

	bt.Save(storage)

	wg.Wait()

	// No streaming
	req.Header.Del("Accept")
	resp, _ := http.DefaultClient.Do(req)
	reader := bufio.NewReader(resp.Body)
	readByte, _ := ioutil.ReadAll(reader)
	var receivedBts BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	assert.Equal(t, bt.Hash, receivedBts.Hash, "hash is not same")

}

func TestGetTransactionsHandler(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)

	storage, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer storage.Close()

	router := mux.NewRouter()
	router.HandleFunc(GetTransactionsHandlerPattern, GetTransactionsHandler(storage)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	var bts []BlockTransaction
	for i := 0; i < 5; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(storage)
		if err != nil {
			t.Error(err)
			return
		}
		bts = append(bts, bt)
	}

	// Do a Request
	url := ts.URL + "/transactions"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, _ := http.DefaultClient.Do(req)
	reader := bufio.NewReader(resp.Body)

	// Do stream Request to the Server
	go func() {
		for n := 0; n < 10; n++ {
			line, _ := reader.ReadBytes('\n')
			line = bytes.Trim(line, "\n\t ")
			txS, _ := bts[n].Serialize()
			if bytes.Compare(txS, line) != 0 {
				t.Error("not same")
			}
		}

		resp.Body.Close()
		wg.Done()
	}()

	for i := 0; i < 20; i++ {
		_, tx := TestMakeTransaction(networkID, 1)

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(storage)
		if err != nil {
			t.Error(err)
			return
		}
		bts = append(bts, bt)
	}

	wg.Wait()

	// No streaming
	req.Header.Del("Accept")
	resp, _ = http.DefaultClient.Do(req)
	reader = bufio.NewReader(resp.Body)
	readByte, _ := ioutil.ReadAll(reader)
	var receivedBts []BlockTransaction
	json.Unmarshal(readByte, &receivedBts)

	assert.Equal(t, len(bts), len(receivedBts), "length is not same")

	i := 0
	for _, bt := range bts {
		assert.Equal(t, bt.Hash, receivedBts[i].Hash, "hash is not same")
		i++
	}

}
