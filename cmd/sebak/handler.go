package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spikeekips/sebak/lib/network"
)

func Index(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO return node info
		fmt.Fprintf(w, "hi~")
	}
}

func MessageHandler(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		t.ReceiveChannel() <- network.TransportMessage{Type: "message", Data: body}

		// TODO return with the link to check the status of message
		return
	}
}

func BallotHandler(t *network.HTTP2Transport) network.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		t.ReceiveChannel() <- network.TransportMessage{Type: "ballot", Data: body}
		return
	}
}
