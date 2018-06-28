package sebak

import (
	"context"
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/GianlucaGuarini/go-observable"
)

const maxNumberOfExistingData = 10

func AddAPIHandlers(s *sebakstorage.LevelDBBackend) func(ctx context.Context, t *sebaknetwork.HTTP2Network) {
	fn := func(ctx context.Context, t *sebaknetwork.HTTP2Network) {
		t.AddAPIHandler(GetAccountHandlerPattern, GetAccountHandler(s)).Methods("GET")
		t.AddAPIHandler(GetAccountTransactionsHandlerPattern, GetAccountTransactionsHandler(s)).Methods("GET")
		t.AddAPIHandler(GetAccountOperationsHandlerPattern, GetAccountOperationsHandler(s)).Methods("GET")
		t.AddAPIHandler(GetTransactionByHashHandlerPattern, GetTransactionByHashHandler(s)).Methods("GET")
	}
	return fn
}

// Implement `Server Sent Event`
// Listen event `event` thru `o`
// When the `event` triggered, `callBackFunc` fired
// readyChan is used to notify caller of this function that streaming is ready
// This function is not end until the connection is closed
func streaming(o *observable.Observable, w http.ResponseWriter, event string, callBackFunc func(args ...interface{}) ([]byte, error), readyChan chan struct{}) {

	cn, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

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
		case <-cn.CloseNotify():
			close(consumerChan)
			return
		case message := <-messageChan:
			fmt.Fprintf(w, "%s\n", message)
			flusher.Flush()
		}
	}

}
