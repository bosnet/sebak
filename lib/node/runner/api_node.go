package runner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

const (
	NodeInfoHandlerPattern string = "/"
	ConnectHandlerPattern  string = "/connect"
	MessageHandlerPattern  string = "/message"
	BallotHandlerPattern   string = "/ballot"
)

type NetworkHandlerNode struct {
	localNode *node.LocalNode
	network   network.Network
	storage   *storage.LevelDBBackend
	consensus *consensus.ISAAC
	urlPrefix string
}

func NewNetworkHandlerNode(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend, consensus *consensus.ISAAC, urlPrefix string) *NetworkHandlerNode {
	return &NetworkHandlerNode{
		localNode: localNode,
		network:   network,
		storage:   storage,
		consensus: consensus,
		urlPrefix: urlPrefix,
	}
}

func (api NetworkHandlerNode) HandlerURLPattern(pattern string) string {
	return fmt.Sprintf("%s%s", api.urlPrefix, pattern)
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
	blk, err := block.GetLatestBlock(api.storage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := NodeInfoWithRequest(api.localNode, &blk, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.network.MessageBroker().Response(w, b)
}

func (api NetworkHandlerNode) ConnectHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.ConnectMessage, Data: body})

	blk, err := block.GetLatestBlock(api.storage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := NodeInfoWithRequest(api.localNode, &blk, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.network.MessageBroker().Response(w, b)
}

func (api NetworkHandlerNode) MessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if ct := r.Header.Get("Content-Type"); strings.ToLower(ct) != "application/json" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.TransactionMessage, Data: body})
	api.network.MessageBroker().Response(w, body)
}

func (api NetworkHandlerNode) BallotHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if ct := r.Header.Get("Content-Type"); strings.ToLower(ct) != "application/json" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	api.network.MessageBroker().Receive(common.NetworkMessage{Type: common.BallotMessage, Data: body})
	api.network.MessageBroker().Response(w, body)

	return
}

func NodeInfoWithRequest(localNode *node.LocalNode, blk *block.Block, r *http.Request) (b []byte, err error) {
	var endpoint string
	if localNode.PublishEndpoint() != nil {
		endpoint = localNode.PublishEndpoint().String()
	} else {
		rUrl := common.RequestURLFromRequest(r)
		rUrl.Path = ""
		rUrl.RawQuery = ""
		endpoint = rUrl.String()
	}

	var blockHeight uint64
	if blk != nil {
		blockHeight = blk.Height
	}

	info := map[string]interface{}{
		"address":      localNode.Address(),
		"alias":        localNode.Alias(),
		"endpoint":     endpoint,
		"state":        localNode.State().String(),
		"validators":   localNode.GetValidators(),
		"block-height": blockHeight,
	}

	b, err = json.Marshal(info)
	return
}
