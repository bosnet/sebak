package runner

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

type NetworkHandlerNode struct {
	localNode *node.LocalNode
	network   network.Network
	storage   *storage.LevelDBBackend
	consensus *consensus.ISAAC
}

func NewNetworkHandlerNode(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend, consensus *consensus.ISAAC) *NetworkHandlerNode {
	return &NetworkHandlerNode{
		localNode: localNode,
		network:   network,
		storage:   storage,
		consensus: consensus,
	}
}

func (api NetworkHandlerNode) renderNodeItem(w http.ResponseWriter, itemType NodeItemDataType, o interface{}) {
	s, err := json.Marshal(o)
	if err != nil {
		itemType = NodeItemError
		s = []byte(err.Error())
	}

	api.writeNodeItem(w, itemType, s)
}

func (api NetworkHandlerNode) writeNodeItem(w http.ResponseWriter, itemType NodeItemDataType, s []byte) {
	w.Write(append([]byte(itemType+" "), append(s, '\n')...))
}

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
		return
	}

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.ConnectMessage, Data: body})
	o, _ := api.localNode.Serialize()
	api.network.MessageBroker().Response(w, o)
}

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

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.TransactionMessage, Data: body})
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

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.BallotMessage, Data: body})
	api.network.MessageBroker().Response(w, body)

	return
}
