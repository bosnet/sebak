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
	node := ctx.Value("node").(util.Serializable)

	return func(w http.ResponseWriter, r *http.Request) {
		o, _ := node.Serialize()
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

		t.ReceiveChannel() <- TransportMessage{Type: MessageTransportMessage, Data: body}

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

		t.ReceiveChannel() <- TransportMessage{Type: BallotTransportMessage, Data: body}
		return
	}
}
