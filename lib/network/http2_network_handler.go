package sebaknetwork

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/common"
)

func NodeInfoHandler(ctx context.Context, t *HTTP2Network) HandlerFunc {
	var currentNode sebakcommon.Serializable

	return func(w http.ResponseWriter, r *http.Request) {
		if currentNode == nil {
			currentNode = ctx.Value("currentNode").(sebakcommon.Serializable)
		}

		o, _ := currentNode.Serialize()
		t.messageBroker.ResponseMessage(w, string(o))
	}
}

func ConnectHandler(ctx context.Context, t *HTTP2Network) HandlerFunc {
	var currentNode sebakcommon.Serializable

	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if currentNode == nil {
			currentNode = ctx.Value("currentNode").(sebakcommon.Serializable)
		}

		if r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}

		t.messageBroker.ReceiveMessage(t, Message{Type: ConnectMessage, Data: body})
		o, _ := currentNode.Serialize()
		t.messageBroker.ResponseMessage(w, string(o))
	}
}

func MessageHandler(ctx context.Context, t *HTTP2Network) HandlerFunc {
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

		t.messageBroker.ReceiveMessage(t, Message{Type: MessageFromClient, Data: body})
		t.messageBroker.ResponseMessage(w, string(body))

		return
	}
}

func BallotHandler(ctx context.Context, t *HTTP2Network) HandlerFunc {
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

		t.messageBroker.ReceiveMessage(t, Message{Type: BallotMessage, Data: body})
		t.messageBroker.ResponseMessage(w, string(body))

		return
	}
}
