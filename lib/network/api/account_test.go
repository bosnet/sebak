package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"testing"

	"boscoin.io/sebak/lib/block"

	"bytes"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
	"strings"
)

func TestGetAccountHandler(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()
	// Make Dummy BlockAccount
	ba := block.TestMakeBlockAccount()
	ba.Save(storage)
	{
		// Do a Request
		url := strings.Replace(GetAccountHandlerPattern, "{id}", ba.Address, -1)
		respBody, err := request(ts, url, false)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.Nil(t, err)
		recv := make(map[string]interface{})
		json.Unmarshal(readByte, &recv)

		require.Equal(t, ba.Address, recv["id"], "hash is not same")
	}
}

func TestGetAccountHandlerStream(t *testing.T) {
	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()
	// Make Dummy BlockAccount
	ba := block.TestMakeBlockAccount()
	{
		// Do a Request
		url := strings.Replace(GetAccountHandlerPattern, "{id}", ba.Address, -1)
		respBody, err := request(ts, url, true)
		require.Nil(t, err)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		{
			ba.Save(storage)
			require.Nil(t, err)
		}

		for {
			line, err := reader.ReadBytes('\n')
			require.Nil(t, err)
			line = bytes.Trim(line, "\n")
			if line == nil {
				continue
			}
			recv := make(map[string]interface{})
			json.Unmarshal(line, &recv)
			require.Equal(t, ba.Address, recv["id"], "hash is not same")
			break
		}
	}
}

// Test that getting an inexisting account returns an error
func TestGetNonExistentAccountHandler(t *testing.T) {

	ts, storage, err := prepareAPIServer()
	require.Nil(t, err)
	defer storage.Close()
	defer ts.Close()
	{
		// Do a Request
		kp, _ := keypair.Random()
		url := strings.Replace(GetAccountHandlerPattern, "{id}", kp.Address(), -1)
		_, err := request(ts, url, false)
		require.NotNil(t, err)
		require.Equal(t, "status code 404 is not 200", err.Error())
	}
}
