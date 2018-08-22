package sebak

import "encoding/json"

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
