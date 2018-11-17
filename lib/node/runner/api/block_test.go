package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/block"
	"github.com/stretchr/testify/require"
)

func TestBlockHAndler(t *testing.T) {
	ts, st := prepareAPIServer()
	defer st.Close()
	defer ts.Close()

	genesis := block.GetLatestBlock(st)

	reqFunc := func(url string) map[string]interface{} {

		respBody := request(ts, url, false)
		defer respBody.Close()
		bs, err := ioutil.ReadAll(bufio.NewReader(respBody))
		require.NoError(t, err)

		result := make(map[string]interface{})
		err = json.Unmarshal(bs, &result)
		require.NoError(t, err)

		return result
	}

	{
		url := strings.Replace(GetBlockHandlerPattern, "{hashOrHeight}", "1", 1)
		res := reqFunc(url)
		require.Equal(t, res["hash"], genesis.Hash)
		require.Equal(t, res["transactions_root"], genesis.TransactionsRoot)
	}

	{
		url := strings.Replace(GetBlockHandlerPattern, "{hashOrHeight}", genesis.Hash, 1)
		res := reqFunc(url)
		require.Equal(t, res["hash"], genesis.Hash)
		require.Equal(t, res["transactions_root"], genesis.TransactionsRoot)
	}

}
