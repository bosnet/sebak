package sebak

import (
	"io/ioutil"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/network"
)

func (api NetworkHandlerNode) MessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		// TODO use http-problem spec
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); strings.ToLower(ct) != "application/json" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}

	api.network.MessageBroker().Receive(sebaknetwork.Message{Type: sebaknetwork.TransactionMessage, Data: body})
	api.network.MessageBroker().Response(w, body)
}

func (api NetworkHandlerNode) BallotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); strings.ToLower(ct) != "application/json" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}

	api.network.MessageBroker().Receive(sebaknetwork.Message{Type: sebaknetwork.BallotMessage, Data: body})
	api.network.MessageBroker().Response(w, body)

	return
}
