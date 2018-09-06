package sebak

import (
	"fmt"
	"net/http"

	"github.com/GianlucaGuarini/go-observable"

	"boscoin.io/sebak/lib/network"
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
)

type NetworkHandlerNode struct {
	localNode *node.LocalNode
	network   network.Network
}

type NetworkHandlerAPI struct {
	localNode *node.LocalNode
	network   network.Network
	storage   *storage.LevelDBBackend
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
