package sebak

import (
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/network"
)

func (api NetworkHandlerNode) NodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	o, _ := api.localNode.Serialize()
	api.network.MessageBroker().Response(w, o)
}

func (api NetworkHandlerNode) ConnectHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}

	api.network.MessageBroker().Receive(sebaknetwork.Message{Type: sebaknetwork.ConnectMessage, Data: body})
	o, _ := api.localNode.Serialize()
	api.network.MessageBroker().Response(w, o)
}
