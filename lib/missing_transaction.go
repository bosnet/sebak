package sebak

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
)

type MissingTxAPI struct {
	noderunner *NodeRunner
	network    *sebaknetwork.Network
}
type MissingTransactions struct {
	MissingTxs map[string]Transaction `json:"txs"`
}

func (m MissingTransactions) Serialize() (result []byte, err error) {
	serializedMap := make(map[string][]byte)
	var txByte []byte
	for hash, tx := range m.MissingTxs {
		txByte, err = json.Marshal(tx)
		if err != nil {
			return
		}
		serializedMap[hash] = txByte
	}
	result, err = json.Marshal(serializedMap)
	return
}

func NewMissingTransactionsFromJSON(b []byte) (*MissingTransactions, error) {
	p := &MissingTransactions{}
	p.MissingTxs = make(map[string]Transaction)

	deserializedMap := make(map[string][]byte)
	err := json.Unmarshal(b, &deserializedMap)

	var tx Transaction
	for hash, txByte := range deserializedMap {
		tx, err = NewTransactionFromJSON(txByte)
		p.MissingTxs[hash] = tx
	}

	return p, err
}

const SendMissingTxToRequestNodePattern = "/missingtx"

func (missingTxApi MissingTxAPI) SendMissingTxToRequestNode(storage *sebakstorage.LevelDBBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		resultTx := make(map[string]Transaction)

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		requestBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		hashesFromRequestNode, err := NewMissingTransactionsFromJSON(requestBody)

		for _, h := range hashesFromRequestNode.MissingTxs {
			hash := h.GetHash()
			if resultTransaction, found := missingTxApi.noderunner.Consensus().TransactionPool.Get(hash); found {
				// append hash to map
				resultTx[hash] = resultTransaction

			} else {
				if found, err := ExistBlockTransaction(storage, hash); err != nil {
					return
				} else if found {
					var missingBlockTransaction BlockTransaction
					if missingBlockTransaction, err = GetBlockTransaction(storage, hash); err != nil {
						return
					}
					// append blocktransaction to Map
					resultTx[hash] = missingBlockTransaction.Transaction()

				}
			}
		}
		missingTxsResult := MissingTransactions{
			MissingTxs: resultTx,
		}
		sendMissingTxsresult, err := missingTxsResult.Serialize()
		if _, err = w.Write(sendMissingTxsresult); err != nil {
			return
		}
	}
}
