package network

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spikeekips/sebak/lib/util"
)

func Index(ctx context.Context, t *HTTP2Transport) HandlerFunc {
	var currentNode util.Serializable

	return func(w http.ResponseWriter, r *http.Request) {
		if currentNode == nil {
			currentNode = ctx.Value("currentNode").(util.Serializable)
		}

		o, _ := currentNode.Serialize()
		fmt.Fprintf(w, string(o))
	}
}

func ConnectHandler(ctx context.Context, t *HTTP2Transport) HandlerFunc {
	var currentNode util.Serializable

	return func(w http.ResponseWriter, r *http.Request) {
		if currentNode == nil {
			currentNode = ctx.Value("currentNode").(util.Serializable)
		}

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}

		// and then connect to remote
		t.ReceiveChannel() <- Message{Type: ConnectMessage, Data: body}

		// send current node info
		o, _ := currentNode.Serialize()
		fmt.Fprintf(w, string(o))
	}
}

func MessageHandler(ctx context.Context, t *HTTP2Transport) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		t.ReceiveChannel() <- Message{Type: MessageFromClient, Data: body}

		// TODO return with the link to check the status of message
		return
	}
}

func BallotHandler(ctx context.Context, t *HTTP2Transport) HandlerFunc {
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

		t.ReceiveChannel() <- Message{Type: BallotMessage, Data: body}
		return
	}
}
