package api

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"strconv"
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

func TestGetAccountsHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	numberOfAccounts := int(DefaultLimit) + 10
	accounts := map[string]*block.BlockAccount{}
	for i := 0; i < numberOfAccounts; i++ {
		ba := block.TestMakeBlockAccount()
		ba.MustSave(storage)
		accounts[ba.Address] = ba
	}

	{ // request with empty body
		respBody := request(ts, GetAccountsHandlerPattern, false, []byte{})
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		require.Equal(t, http.StatusBadRequest, int(recv["status"].(float64)))
	}

	{ // request with empty list
		b := common.MustJSONMarshal([]string{})
		respBody := request(ts, GetAccountsHandlerPattern, false, b)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		require.Equal(t, http.StatusBadRequest, int(recv["status"].(float64)))
		require.True(
			t,
			strings.HasSuffix(
				recv["type"].(string),
				strconv.FormatUint(uint64(errors.BadRequestParameter.Code), 10),
			),
		)
	}

	{ // request with addresses
		var expectedAddresses []string
		for address, _ := range accounts {
			expectedAddresses = append(expectedAddresses, address)
			if len(expectedAddresses) == int(DefaultLimit) {
				break
			}
		}

		b := common.MustJSONMarshal(expectedAddresses)
		respBody := request(ts, GetAccountsHandlerPattern, false, b)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)

		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		require.Equal(t, len(expectedAddresses), len(records))
		for _, r := range records {
			o := r.(map[string]interface{})
			address := o["address"].(string)
			require.NotEmpty(t, accounts[address])
			require.Equal(t, accounts[address].Balance, common.MustAmountFromString(o["balance"].(string)))
		}
	}

	{ // request with over limit
		var expectedAddresses []string
		for address, _ := range accounts {
			expectedAddresses = append(expectedAddresses, address)
		}

		b := common.MustJSONMarshal(expectedAddresses)
		respBody := request(ts, GetAccountsHandlerPattern, false, b)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)

		require.Equal(t, http.StatusBadRequest, int(recv["status"].(float64)))
		require.True(
			t,
			strings.HasSuffix(
				recv["type"].(string),
				strconv.FormatUint(uint64(errors.PageQueryLimitMaxExceed.Code), 10),
			),
		)
	}

	{ // request with unknown addresses; the unknown address will not be included in the response
		var expectedAddresses []string
		for address, _ := range accounts {
			expectedAddresses = append(expectedAddresses, address)
			if len(expectedAddresses) == int(DefaultLimit)-2 {
				break
			}
		}

		unknownAddresses := []string{
			keypair.Random().Address(),
			keypair.Random().Address(),
		}
		expectedAddresses = append(expectedAddresses, unknownAddresses...)

		b := common.MustJSONMarshal(expectedAddresses)
		respBody := request(ts, GetAccountsHandlerPattern, false, b)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)

		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		require.Equal(t, len(expectedAddresses)-len(unknownAddresses), len(records))
		for _, r := range records {
			o := r.(map[string]interface{})
			address := o["address"].(string)
			require.NotEmpty(t, accounts[address])
			require.Equal(t, accounts[address].Balance, common.MustAmountFromString(o["balance"].(string)))
		}
	}

}
