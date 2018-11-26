package api

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction/operation"
	"bufio"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"strings"
	"testing"
)

func TestGetOperationsByAccountHandler(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, kpTarget, boList := prepareOps(storage, 10)

	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	{
		ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}
	{
		ba := block.NewBlockAccount(kpTarget.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}

	// Do a Request for source account
	{
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")
		}
	}

	// Do a Request for target account
	url = strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kpTarget.Address(), -1)
	{
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")

			blk, _ := block.GetBlockByHeight(storage, uint64(bt["block_height"].(float64)))
			require.Equal(t, blk.ProposedTime, bt["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, bt["confirmed"].(string))
		}
	}
}

func TestGetOperationsByAccountHandlerWithType(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, kpTarget, boList := prepareOps(storage, 10)
	{
		ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}
	{
		ba := block.NewBlockAccount(kpTarget.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}

	// Do a Request for Source
	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	{
		url := url + "?type=" + string(operation.TypeCreateAccount)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"]
		require.Nil(t, records)
	}

	{
		url := url + "?type=" + string(operation.TypePayment)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")

			blk, _ := block.GetBlockByHeight(storage, uint64(bt["block_height"].(float64)))
			require.Equal(t, blk.ProposedTime, bt["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, bt["confirmed"].(string))
		}
	}

	// Do a Request for Target
	url = strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kpTarget.Address(), -1)
	{
		url := url + "?type=" + string(operation.TypeCreateAccount)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"]
		require.Nil(t, records)
	}

	{
		url := url + "?type=" + string(operation.TypePayment)
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)

		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})

		require.Equal(t, len(boList), len(records), "length is not same")

		for i, r := range records {
			bt := r.(map[string]interface{})
			hash := bt["hash"].(string)

			require.Equal(t, hash, boList[i].Hash, "hash is not same")

			blk, _ := block.GetBlockByHeight(storage, uint64(bt["block_height"].(float64)))
			require.Equal(t, blk.ProposedTime, bt["proposed_time"].(string))
			require.Equal(t, blk.Confirmed, bt["confirmed"].(string))
		}
	}

}

func TestGetOperationsByAccountHandlerPage(t *testing.T) {
	ts, storage := prepareAPIServer()
	defer storage.Close()
	defer ts.Close()

	kp, kpTarget, boList := prepareOps(storage, 50)

	url := strings.Replace(GetAccountOperationsHandlerPattern, "{id}", kp.Address(), -1)
	url += "?limit=20"
	{
		ba := block.NewBlockAccount(kp.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}
	{
		ba := block.NewBlockAccount(kpTarget.Address(), common.Amount(common.BaseReserve))
		ba.MustSave(storage)
	}

	for i, bo := range boList {
		t.Log(i, bo.Hash)
	}

	// Do a Request for source account
	prev := ""
	next := ""

	// 0 ~ 19
	{
		respBody := request(ts, url, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		prev = recv["_links"].(map[string]interface{})["prev"].(map[string]interface{})["href"].(string)
		next = recv["_links"].(map[string]interface{})["next"].(map[string]interface{})["href"].(string)
		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, boList[i].Hash, bt["hash"], "hash is not same")
		}
	}

	// 20 ~ 39
	{
		respBody := request(ts, next, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		prev = recv["_links"].(map[string]interface{})["prev"].(map[string]interface{})["href"].(string)
		next = recv["_links"].(map[string]interface{})["next"].(map[string]interface{})["href"].(string)
		t.Log(prev)
		t.Log(next)
		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, boList[i+20].Hash, bt["hash"], "hash is not same")
		}

	}

	// 40 ~ 49
	{
		respBody := request(ts, next, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		prev = recv["_links"].(map[string]interface{})["prev"].(map[string]interface{})["href"].(string)
		next = recv["_links"].(map[string]interface{})["next"].(map[string]interface{})["href"].(string)
		t.Log(prev)
		t.Log(next)
		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, boList[i+40].Hash, bt["hash"], "hash is not same")
		}
	}

	// 40 ~ 21
	{
		respBody := request(ts, prev, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		prev = recv["_links"].(map[string]interface{})["prev"].(map[string]interface{})["href"].(string)
		next = recv["_links"].(map[string]interface{})["next"].(map[string]interface{})["href"].(string)
		t.Log(prev)
		t.Log(next)
		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, boList[40-i].Hash, bt["hash"], "hash is not same")
		}
	}

	// 20 ~ 1
	{
		respBody := request(ts, prev, false)
		defer respBody.Close()
		reader := bufio.NewReader(respBody)
		readByte, err := ioutil.ReadAll(reader)
		require.NoError(t, err)

		recv := make(map[string]interface{})
		common.MustUnmarshalJSON(readByte, &recv)
		records := recv["_embedded"].(map[string]interface{})["records"].([]interface{})
		prev = recv["_links"].(map[string]interface{})["prev"].(map[string]interface{})["href"].(string)
		next = recv["_links"].(map[string]interface{})["next"].(map[string]interface{})["href"].(string)
		t.Log(prev)
		t.Log(next)
		for i, r := range records {
			bt := r.(map[string]interface{})
			require.Equal(t, boList[20-i].Hash, bt["hash"], "hash is not same")
		}
	}
}
