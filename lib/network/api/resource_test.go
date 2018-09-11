package api

import (
	"encoding/json"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stretchr/testify/require"
)

func TestAPIResourceAccount(t *testing.T) {
	storage, err := storage.NewTestMemoryLevelDBBackend()
	require.Nil(t, err)
	defer storage.Close()

	// Account
	{
		ba := block.TestMakeBlockAccount()
		ba.Save(storage)
		ra := &APIResourceAccount{
			accountId:  ba.Address,
			sequenceID: ba.SequenceID,
			balance:    ba.GetBalance().String(),
		}
		r := ra.Resource()
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, ba.Address, m["account_id"])
			require.Equal(t, ba.Address, m["id"])
			require.Equal(t, ba.SequenceID, uint64(m["sequence_id"].(float64)))
			require.Equal(t, ba.GetBalance().String(), m["balance"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlAccounts, "{id}", ba.Address, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// Transaction
	{
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 1)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction("dummy", tx, a)
		bt.Save(storage)

		rt := &APIResourceTransaction{
			hash:       bt.Hash,
			sequenceID: bt.SequenceID,
			signature:  bt.Signature,
			source:     bt.Source,
			fee:        bt.Fee.String(),
			amount:     bt.Amount.String(),
			created:    bt.Created,
			operations: bt.Operations,
		}
		r := rt.Resource()
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bt.Hash, m["id"])
			require.Equal(t, bt.Hash, m["hash"])
			require.Equal(t, bt.Source, m["account"])
			require.Equal(t, bt.Fee.String(), m["fee_paid"])
			require.Equal(t, bt.Created, m["created_at"])
			require.Equal(t, float64(len(bt.Operations)), m["operation_count"])

			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlTransactions, "{id}", bt.Hash, -1), l["self"].(map[string]interface{})["href"])
		}

	}

	// Operation
	{
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 1)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction("dummy", tx, a)
		bt.Save(storage)
		bo, err := block.GetBlockOperation(storage, bt.Operations[0])

		ro := &APIResourceOperation{
			hash:    bo.Hash,
			txHash:  bo.TxHash,
			funder:  bo.Source,
			account: bo.Target,
			otype:   string(bo.Type),
			amount:  bo.Amount.String(),
		}
		r := ro.Resource()
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{
			var f interface{}
			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})
			require.Equal(t, bo.Hash, m["id"])
			require.Equal(t, bo.Hash, m["hash"])
			require.Equal(t, bo.Source, m["funder"])
			require.Equal(t, bo.Target, m["account"])
			require.Equal(t, string(bo.Type), m["type"])
			require.Equal(t, bo.Amount.String(), m["amount"])
			l := m["_links"].(map[string]interface{})
			require.Equal(t, strings.Replace(UrlOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
		}
	}

	// List
	{
		_, tx := transaction.TestMakeTransaction([]byte{0x00}, 3)
		a, err := tx.Serialize()
		require.Nil(t, err)
		bt := block.NewBlockTransactionFromTransaction("dummy", tx, a)
		bt.Save(storage)

		var rol []APIResource
		for _, boHash := range bt.Operations {
			var bo block.BlockOperation
			bo, err = block.GetBlockOperation(storage, boHash)
			require.Nil(t, err)

			ro := &APIResourceOperation{
				hash:    bo.Hash,
				txHash:  bo.TxHash,
				funder:  bo.Source,
				account: bo.Target,
				otype:   string(bo.Type),
				amount:  bo.Amount.String(),
			}
			rol = append(rol, ro)
		}

		urlneedToBeFilledByAPI := "/operations/"
		arl := &APIResourceList{Resources: rol, SelfLink: urlneedToBeFilledByAPI}
		r := arl.Resource()
		j, _ := json.MarshalIndent(r, "", " ")
		//fmt.Printf("%s\n", j)

		{

			var f interface{}

			json.Unmarshal(j, &f)
			m := f.(map[string]interface{})

			l := m["_links"].(map[string]interface{})
			require.Equal(t, urlneedToBeFilledByAPI, l["self"].(map[string]interface{})["href"])

			records := m["_embedded"].(map[string]interface{})["records"].([]interface{})
			for _, v := range records {
				record := v.(map[string]interface{})
				id := record["id"].(string)
				bo, err := block.GetBlockOperation(storage, id)
				require.Nil(t, err)
				require.Equal(t, bo.Hash, record["id"])
				require.Equal(t, bo.Hash, record["hash"])
				require.Equal(t, bo.Source, record["funder"])
				require.Equal(t, bo.Target, record["account"])
				require.Equal(t, string(bo.Type), record["type"])
				require.Equal(t, bo.Amount.String(), record["amount"])
				l := record["_links"].(map[string]interface{})
				require.Equal(t, strings.Replace(UrlOperations, "{id}", bo.Hash, -1), l["self"].(map[string]interface{})["href"])
			}
		}
	}
}
