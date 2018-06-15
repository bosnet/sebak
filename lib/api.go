package sebak

import (
	"context"
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/network"
	"github.com/GianlucaGuarini/go-observable"
)

func AddAPIHandlers(ctx context.Context, t *sebaknetwork.HTTP2Network) {

	t.AddAPIHandler(ctx, "/account/{address}", GetAccountHandler).Methods("GET")
}

func streaming(o *observable.Observable, w http.ResponseWriter, event string, callBackFunc func(args ...interface{}) ([]byte, error), once []byte) {
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

	closeChan := make(chan bool)
	messageChan := make(chan []byte)

	observerFunc := func(args ...interface{}) {
		s, err := callBackFunc(args...)
		if err != nil {
			closeChan <- true
			return
		}
		messageChan <- s
	}

	o.On(event, observerFunc)
	defer o.Off(event, observerFunc)

	w.Header().Set("Content-Type", "application/json")

	if len(once) != 0 {
		fmt.Fprintf(w, "%s\n", once)
		flusher.Flush()
	}

	for {
		select {
		case <-cn.CloseNotify():
			return
		case <-closeChan:
			return
		case message := <-messageChan:
			fmt.Fprintf(w, "%s\n", message)
			flusher.Flush()
		}
	}
}
