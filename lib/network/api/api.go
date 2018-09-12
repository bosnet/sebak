package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GianlucaGuarini/go-observable"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

const maxNumberOfExistingData = 10

// API Endpoint patterns
const (
	GetAccountTransactionsHandlerPattern = "/account/{address}/transactions"
	GetAccountHandlerPattern             = "/account/{address}"
	GetAccountOperationsHandlerPattern   = "/account/{address}/operations"
	GetTransactionsHandlerPattern        = "/transactions"
	GetTransactionByHashHandlerPattern   = "/transactions/{txid}"
	PostTransactionPattern               = "/transactions"
)

type NetworkHandlerAPI struct {
	localNode *node.LocalNode
	network   network.Network
	storage   *storage.LevelDBBackend
}

func NewNetworkHandlerAPI(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend) *NetworkHandlerAPI {
	return &NetworkHandlerAPI{
		localNode: localNode,
		network:   network,
		storage:   storage,
	}
}

func renderEventStream(args ...interface{}) ([]byte, error) {
	if len(args) <= 1 {
		return nil, fmt.Errorf("render: value is empty") //TODO(anarcher): Error type
	}
	i := args[1]

	switch v := i.(type) {
	case *block.BlockAccount:
		r := resource.NewAccount(v)
		return json.Marshal(r.Resource())
	case *block.BlockOperation:
		r := resource.NewOperation(v)
		return json.Marshal(r.Resource())
	case *block.BlockTransaction:
		r := resource.NewTransaction(v)
		return json.Marshal(r.Resource())
	case httputils.HALResource:
		return json.Marshal(v.Resource())
	}

	return json.Marshal(i)
}

// Implement `Server Sent Event`
// Listen event `event` thru `o`
// When the `event` triggered, `callBackFunc` fired
// readyChan is used to notify caller of this function that streaming is ready
// This function is not end until the connection is closed
func streaming(o *observable.Observable, r *http.Request, w http.ResponseWriter, event string, callBackFunc func(args ...interface{}) ([]byte, error), readyChan chan struct{}) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// consumerChan notify observerFunc that messageChan receiver is dismissed
	consumerChan := make(chan struct{})
	messageChan := make(chan []byte)

	observerFunc := func(args ...interface{}) {
		s, err := callBackFunc(args...)
		if err != nil {
			//TODO: handle the error
			return
		}

		select {
		case messageChan <- s:
		case <-consumerChan:
		}
	}

	o.On(event, observerFunc)
	defer o.Off(event, observerFunc)

	w.Header().Set("Content-Type", "application/json")

	readyChan <- struct{}{}
	for {
		select {
		case <-r.Context().Done():
			close(consumerChan)
			return
		case message := <-messageChan:
			fmt.Fprintf(w, "%s\n", message)
			flusher.Flush()
		}
	}
}
