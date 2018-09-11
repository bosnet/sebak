package api

import (
	"bufio"
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
		cba := map[string]interface{}{}
		json.Unmarshal(line, &cba)
		balance := common.MustAmountFromString(cba["balance"].(string))

		require.Equal(t, ba.Address, cba["account_id"])
		require.Equal(t, prev+n, balance)

		prev = balance
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
	cba := map[string]interface{}{}
	json.Unmarshal(readByte, &cba)
	balance := common.MustAmountFromString(cba["balance"].(string))

	require.Equal(t, ba.Address, cba["account_id"], "not equal")
	require.Equal(t, ba.GetBalance(), balance, "not equal")
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
