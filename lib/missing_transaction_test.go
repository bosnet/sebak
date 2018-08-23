package sebak

import (
	"encoding/json"
	//"fmt"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	//"net/http"
	"testing"

	"boscoin.io/sebak/lib/common"
)

func TestTransactionSerializeDeserializeClient(t *testing.T) {
	transactions := generateTransactionsSlice(t, 3)
	result, err := json.Marshal(transactions)
	assert.Nil(t, err)

	var results []string

	err = json.Unmarshal(result, &results)
	assert.Nil(t, err)

	assert.Equal(t, 3, len(results))
}

func TestTransactionSerializeDeserializeServer(t *testing.T) {
	transactions := generateTransactionMap(t, 3)
	mtxs := MissingTransactions{
		MissingTxs: transactions,
	}
	serialized, err := mtxs.Serialize()
	assert.Nil(t, err)

	deserialized, err := NewMissingTransactionsFromJSON(serialized)
	assert.Nil(t, err)

	assert.Equal(t, 3, len(deserialized.MissingTxs))
	for hash, tx := range transactions {
		assert.Equal(t, deserialized.MissingTxs[hash].GetHash(), tx.GetHash())
	}
}

func generateTransactionMap(t *testing.T, n int) (txs map[string]Transaction) {
	txs = make(map[string]Transaction)
	for i := 0; i < 3; i++ {
		sender, _ := keypair.Random()
		receiver, _ := keypair.Random()

		initialBalance := sebakcommon.Amount(1)

		tx := makeTransactionCreateAccount(sender, receiver.Address(), initialBalance)
		tx.B.Checkpoint = sebakcommon.MakeGenesisCheckpoint(networkID)
		tx.Sign(sender, networkID)

		txByte, err := tx.Serialize()
		assert.Nil(t, err)

		deserializedTx, err := NewTransactionFromJSON(txByte)
		assert.Nil(t, err)

		assert.Equal(t, tx.GetHash(), deserializedTx.GetHash())
		txs[tx.GetHash()] = tx
	}
	return
}

func generateTransactionsSlice(t *testing.T, n int) (txs []string) {
	for i := 0; i < 3; i++ {
		sender, _ := keypair.Random()
		receiver, _ := keypair.Random()

		initialBalance := sebakcommon.Amount(1)

		tx := makeTransactionCreateAccount(sender, receiver.Address(), initialBalance)
		tx.B.Checkpoint = sebakcommon.MakeGenesisCheckpoint(networkID)
		tx.Sign(sender, networkID)

		txByte, err := tx.Serialize()
		assert.Nil(t, err)

		deserializedTx, err := NewTransactionFromJSON(txByte)
		assert.Nil(t, err)

		assert.Equal(t, tx.GetHash(), deserializedTx.GetHash())
		txs = append(txs, tx.GetHash())
	}
	return
}

func TestHTTP2NetworkGetTransactions(t *testing.T) {
	// var transactions []byte
	//_, s0, _ := sebaknetwork.CreateNewHTTP2Network(t)
	//s0.SetMessageBroker(sebaknetwork.TestMessageBroker{})
	// s0.Ready()

	// go s0.Start()
	// defer s0.Stop()

	// c0 := s0.GetClient(s0.Endpoint())
	// //sebaknetwork.PingAndWait(t, c0)

	// returnMsg, _ := GetTransactions(transactions)

	// assert.Equal(t, returnStr, sendMsg, "The sendMessage and the return should be the same.")
}

func GetTransactions() {
	return
}
