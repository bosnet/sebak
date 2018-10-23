package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestUnmarshaAccount(t *testing.T) {

	storage := storage.NewTestStorage()
	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")

	ts := httptest.NewServer(router)
	var err error
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()

	ba := block.TestMakeBlockAccount()
	ba.SequenceID = uint64(1)
	if err := ba.Save(storage); err != nil {
		panic(err)
	}

	{
		// Do a Request
		url := strings.Replace(GetAccountHandlerPattern, "{id}", ba.Address, -1)
		respBody, err := request(ts, url, false)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		var ba2 block.BlockAccount
		json.Unmarshal(readByte, &ba2)

		require.Equal(t, ba.SequenceID, ba2.SequenceID, "address is not same")
	}
}

func (api NetworkHandlerAPI) GetTestAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.ErrorBlockAccountDoesNotExists
		}
		ba, err := block.GetBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		payload = resource.NewAccount(ba)
		return payload, nil
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("address-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		}
		es.Run(observer.BlockAccountObserver, event)
		return
	}

	payload, err := readFunc()
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	if err := httputils.WriteJSON(w, 200, payload); err != nil {
		httputils.WriteJSONError(w, err)
	}
}
