package api

/*
import (
	"io/ioutil"
	"boscoin.io/sebak/lib/network/httputils"
	"github.com/gorilla/mux"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/common/observer"
)

import (
	"io/ioutil"
	"boscoin.io/sebak/lib/network/httputils"
	"github.com/gorilla/mux"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/common/observer"
	"net/http"
	"fmt"
	"boscoin.io/sebak/lib/error"
)

func (api NetworkHandlerAPI) PostTransactionByHashHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	message := common.NetworkMessage{Type: common.TransactionMessage, Data: body}
	checker := &MessageChecker{
		DefaultChecker: common.DefaultChecker{Funcs: HandleTransactionCheckerFuncs},
		Consensus:      api.consensus,
		Storage:        api.storage,
		LocalNode:      api.localNode,
		NetworkID:      api.consensus.NetworkID,
		Message:        message,
		Log:            log,
		Conf:           api.conf,
	}

	if err = common.RunChecker(checker, common.DefaultDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); ok {
			return
		}
		httputils.WriteJSONError(w, err)
		return
	}
}
func (api NetworkHandlerAPI) GetTransactionByHashHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockTransaction(api.storage, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.ErrorBlockTransactionDoesNotExists
		}
		bt, err := block.GetBlockTransaction(api.storage, key)
		if err != nil {
			return nil, err
		}
		payload = resource.NewTransaction(&bt)
		return payload, nil
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("hash-%s", key)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}
	payload, err := readFunc()
	if err == nil {
		if err := httputils.WriteJSON(w, 200, payload); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}
	} else {
		if err := httputils.WriteJSON(w, httputils.StatusCode(err), err); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}
	}
}
*/