package runner

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

const GetTransactionPattern string = "/transactions"

func (nh NetworkHandlerNode) GetNodeTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	hashes := r.URL.Query()["hash"]
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, errors.ErrorContentTypeNotJSON.Error(), http.StatusBadRequest)
			return
		}

		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		if len(body) > 0 {
			var postHashes []string
			if err := json.Unmarshal(body, &postHashes); err != nil {
				http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
				return
			}

			hashes = append(hashes, postHashes...)
		}
	}
	if len(hashes) < 1 {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	// Usually `GetNodeTransactionsHandler` will be used for finding the missing
	// `Transaction`s from proposer, so it can not be over the maximum number of
	// `Transaction`s in one `Ballot`.
	if len(hashes) > common.MaxTransactionsInBallot {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-SEBAK-RESULT-COUNT", string(len(hashes)))

	unknown := map[string]struct{}{}

	// check in `TransactionPool`
	for _, hash := range hashes {
		if tx, found := nh.consensus.TransactionPool.Get(hash); !found {
			unknown[hash] = struct{}{}
			continue
		} else {
			nh.renderNodeItem(w, NodeItemTransaction, tx)
		}
	}

	// check in `BlockTransaction`
	for _, hash := range hashes {
		if _, found := unknown[hash]; !found {
			continue
		}

		if exists, err := block.ExistBlockTransaction(nh.storage, hash); err != nil {
			nh.renderNodeItem(w, NodeItemError, err)
			return
		} else if !exists {
			nh.renderNodeItem(w, NodeItemError, errors.ErrorTransactionNotFound.Clone().SetData("hash", hash))
			continue
		}

		btx, err := block.GetBlockTransaction(nh.storage, hash)
		if err != nil {
			nh.renderNodeItem(w, NodeItemError, err)
			return
		}
		nh.writeNodeItem(w, NodeItemTransaction, btx.Message)
	}

	return
}
