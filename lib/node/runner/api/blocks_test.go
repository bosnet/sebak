package api

import (
	"bufio"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
)

func TestBlocksHandler(t *testing.T) {
	ts, st := prepareAPIServer()
	defer st.Close()
	defer ts.Close()

	const numBlock = 10

	genesis := block.GetLatestBlock(st)
	inserted := []block.Block{genesis}
	for i := genesis.Height + 1; i <= numBlock; i++ {
		bk := block.TestMakeNewBlockWithPrevBlock(inserted[len(inserted)-1], []string{})
		bk.MustSave(st)
		inserted = append(inserted, bk)
	}

	reqFunc := func(url string) ([]interface{}, map[string]interface{}) {
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		bs, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		result := make(map[string]interface{})
		common.MustUnmarshalJSON(bs, &result)
		records := result["_embedded"].(map[string]interface{})["records"].([]interface{})
		links := result["_links"].(map[string]interface{})
		return records, links
	}

	testFunc := func(query string) ([]interface{}, map[string]interface{}) {
		return reqFunc(GetBlocksHandlerPattern + "?" + query)
	}

	{
		q := "cursor=1&limit=10&reverse=false"
		records, _ := testFunc(q)

		require.Equal(t, len(records), 10)
		for i, a := range inserted {
			b := records[i].(map[string]interface{})
			require.Equal(t, a.Hash, b["hash"])
		}
	}

	{
		q := "cursor=10&limit=10&reverse=true"
		records, _ := testFunc(q)
		require.Equal(t, 10, len(records))
		for i, a := range inserted {
			b := records[9-i].(map[string]interface{})
			require.Equal(t, a.Hash, b["hash"])
		}
	}
}
