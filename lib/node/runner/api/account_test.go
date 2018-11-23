package api

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"

	"github.com/stretchr/testify/require"
)

func TestGetAccountHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()
	// Make Dummy BlockAccount
	ba := block.TestMakeBlockAccount()
	ba.MustSave(storage)
	{
		// Do a Request
		url := strings.Replace(GetAccountHandlerPattern, "{id}", ba.Address, -1)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)

		require.Equal(t, ba.Address, recv["address"], "address is not same")
	}

	{ // unknown address
		unknownKey := keypair.Random()
		url := strings.Replace(GetAccountHandlerPattern, "{id}", unknownKey.Address(), -1)
		req, _ := http.NewRequest("GET", ts.URL+url, nil)
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	}
}

// Test that getting an inexisting account returns an error
func TestGetNonExistentAccountHandler(t *testing.T) {

	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	p := httputils.NewErrorProblem(errors.BlockAccountDoesNotExists, httputils.StatusCode(errors.BlockAccountDoesNotExists))

	{
		// Do a Request
		url := strings.Replace(GetAccountHandlerPattern, "{id}", keypair.Random().Address(), -1)
		respBody := request(ts, url, false)
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		pByte := common.MustJSONMarshal(p)
		require.Equal(t, pByte, readByte)
	}
}
