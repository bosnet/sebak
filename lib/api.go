package sebak

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"github.com/GianlucaGuarini/go-observable"
	"github.com/gorilla/mux"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request)

func AddHandlers(r interface{}, ctx context.Context) {
	if r == nil {
		return
	}
	router, ok := r.(*mux.Router)
	if !ok {
		return
	}
	router.HandleFunc("/account/{address}", GetAccountHandler(ctx)).Methods("GET")
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

	w.Header().Set("Content-Type", "application/json")

	if len(once) != 0 {
		fmt.Fprintf(w, "%s\n", once)
		flusher.Flush()
	}

	for {
		select {
		case <-cn.CloseNotify():
			o.Off(event, observerFunc)
			return
		case <-closeChan:
			o.Off(event, observerFunc)
			return
		case message := <-messageChan:
			fmt.Fprintf(w, "%s\n", message)
			flusher.Flush()
		}
	}
}

func GetAccountHandler(ctx context.Context) HandlerFunc {
	storage, ok := ctx.Value("storage").(*sebakstorage.LevelDBBackend)
	if !ok {
		panic(errors.New("storage is missing in context"))
	}

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		address := vars["address"]
		if found, err := ExistBlockAccount(storage, address); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		} else if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var err error

		var ba *BlockAccount
		if ba, err = GetBlockAccount(storage, address); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		var s []byte
		if s, err = ba.Serialize(); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			event := fmt.Sprintf("saved-%s", address)
			callBackFunc := func(args ...interface{}) (account []byte, err error) {
				ba := args[0].(*BlockAccount)
				if account, err = ba.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockAccountDoesNotExists
				}
				return account, nil
			}
			streaming(BlockAccountObserver, w, event, callBackFunc, s)
		default:
			if _, err = w.Write(s); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}
