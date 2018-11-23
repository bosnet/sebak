package resource

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

func TestResourceAccount(t *testing.T) {
	storage := storage.NewTestStorage()
	defer storage.Close()

	blk := block.TestMakeNewBlockWithPrevBlock(block.Block{}, []string{})
	blk.MustSave(storage)

	// Account
	{
		ba := block.TestMakeBlockAccount()
		ba.SequenceID = 123
		ba.MustSave(storage)

		ra := NewAccount(ba)
		r := ra.Resource()
		j, _ := json.MarshalIndent(r, "", " ")

		{
			var f interface{}
			common.MustUnmarshalJSON(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, ba.Address, m["address"])
			require.Equal(t, ba.SequenceID, uint64(m["sequence_id"].(float64)))
			require.Equal(t, ba.GetBalance().String(), m["balance"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(URLAccounts, "{id}", ba.Address, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// Transaction
	{
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 1)
		bt := block.NewBlockTransactionFromTransaction("dummy", 0, common.NowISO8601(), tx)
		bt.MustSave(storage)

		rt := NewTransaction(&bt)
		r := rt.Resource()
		j, _ := json.MarshalIndent(r, "", " ")

		{
			var f interface{}
			common.MustUnmarshalJSON(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bt.Hash, m["hash"])
			require.Equal(t, bt.Source, m["source"])
			require.Equal(t, bt.Fee.String(), m["fee"])
			require.Equal(t, bt.Created, m["created"])
			require.Equal(t, float64(len(bt.Operations)), m["operation_count"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(URLTransactions, "{id}", bt.Hash, -1), l["self"].(map[string]interface{})["href"])
		}

	}

	// Operation
	{
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 1)
		bt := block.NewBlockTransactionFromTransaction(common.GetUniqueIDFromUUID(), 0, common.NowISO8601(), tx)
		bt.MustSave(storage)
		bo, _ := block.GetBlockOperation(storage, bt.Operations[0])

		ro := NewOperation(&bo)
		r := ro.Resource()
		j, _ := json.MarshalIndent(r, "", " ")

		{
			var f interface{}
			common.MustUnmarshalJSON(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bo.Hash, m["hash"])
			require.Equal(t, bo.Source, m["source"])
			require.Equal(t, string(bo.Type), m["type"])
			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(URLOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// List
	{
		var err error
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 3)
		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, common.NowISO8601(), tx)
		bt.MustSave(storage)
		err = bt.SaveBlockOperations(storage)
		require.NoError(t, err)

		var rol []Resource
		for _, boHash := range bt.Operations {
			var bo block.BlockOperation
			bo, err = block.GetBlockOperation(storage, boHash)
			require.NoError(t, err)

			ro := NewOperation(&bo)
			rol = append(rol, ro)
		}

		urlneedToBeFilledByAPI := "/operations/"
		arl := NewResourceList(rol, urlneedToBeFilledByAPI, urlneedToBeFilledByAPI, urlneedToBeFilledByAPI)
		r := arl.Resource()
		j, _ := json.MarshalIndent(r, "", " ")

		{

			var f interface{}

			common.MustUnmarshalJSON(j, &f)
			m := f.(map[string]interface{})

			l := m["_links"].(map[string]interface{})
			require.Equal(t, urlneedToBeFilledByAPI, l["self"].(map[string]interface{})["href"])

			records := m["_embedded"].(map[string]interface{})["records"].([]interface{})
			for _, v := range records {
				record := v.(map[string]interface{})
				id := record["hash"].(string)
				bo, err := block.GetBlockOperation(storage, id)
				require.NoError(t, err)
				require.Equal(t, bo.Hash, record["hash"])
				require.Equal(t, bo.Source, record["source"])
				require.Equal(t, string(bo.Type), record["type"])
				l := record["_links"].(map[string]interface{})
				require.Equal(t, strings.Replace(URLOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
			}
		}
	}
}
