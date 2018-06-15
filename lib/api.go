package sebak

import (
	"context"
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/GianlucaGuarini/go-observable"
)

func AddAPIHandlers(s *sebakstorage.LevelDBBackend) func(ctx context.Context, t *sebaknetwork.HTTP2Network) {
	fn := func(ctx context.Context, t *sebaknetwork.HTTP2Network) {
		t.AddAPIHandler(GetAccountHandlerPattern, GetAccountHandler(s)).Methods("GET")
		t.AddAPIHandler(GetAccountTransactionsHandlerPattern, GetAccountTransactionsHandler(s)).Methods("GET")
		t.AddAPIHandler(GetAccountOperationsHandlerPattern, GetAccountOperationsHandler(s)).Methods("GET")
		t.AddAPIHandler(GetTransactionByHashHandlerPattern, GetTransactionByHashHandler(s)).Methods("GET")
	}
	return fn
}

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
