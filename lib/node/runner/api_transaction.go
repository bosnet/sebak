package runner

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
)

const GetTransactionPattern string = "/transactions"

func (nh NetworkHandlerNode) GetNodeTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	hashes := r.URL.Query()["hash"]
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, errors.ContentTypeNotJSON.Error(), http.StatusBadRequest)
			return
		}

		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		if len(body) > 0 {
			var postHashes []string
			if err := json.Unmarshal(body, &postHashes); err != nil {
				http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
				return
			}

			hashes = append(hashes, postHashes...)
		}
	}
	if len(hashes) < 1 {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	// Usually `GetNodeTransactionsHandler` will be used for finding the missing
	// `Transaction`s from proposer, so it can not be over the maximum number of
	// `Transaction`s in one `Ballot`.
	if len(hashes) > nh.consensus.Conf.TxsLimit {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SEBAK-RESULT-COUNT", string(len(hashes)))

	// check in `block.TransactionPool`
	for _, hash := range hashes {
		if exists, err := block.ExistsTransactionPool(nh.storage, hash); err != nil {
			nh.renderNodeItem(w, NodeItemError, err)
			return
		} else if !exists {
			nh.renderNodeItem(w, NodeItemError, errors.TransactionNotFound.Clone().SetData("hash", hash))
			continue
		}

		btx, err := block.GetTransactionPool(nh.storage, hash)
		if err != nil {
			nh.renderNodeItem(w, NodeItemError, err)
			return
		}
		nh.writeNodeItem(w, NodeItemTransaction, btx.Message)
	}

	return
}
