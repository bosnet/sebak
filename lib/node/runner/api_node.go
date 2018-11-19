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
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

const (
	NodeInfoHandlerPattern string = "/"
	ConnectHandlerPattern  string = "/connect"
	MessageHandlerPattern  string = "/message"
	BallotHandlerPattern   string = "/ballot"
)

type NetworkHandlerNode struct {
	localNode       *node.LocalNode
	network         network.Network
	storage         *storage.LevelDBBackend
	consensus       *consensus.ISAAC
	transactionPool *transaction.Pool
	urlPrefix       string
	conf            common.Config
}

func NewNetworkHandlerNode(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend, consensus *consensus.ISAAC, transactionPool *transaction.Pool, urlPrefix string, conf common.Config) *NetworkHandlerNode {
	return &NetworkHandlerNode{
		localNode:       localNode,
		network:         network,
		storage:         storage,
		consensus:       consensus,
		transactionPool: transactionPool,
		urlPrefix:       urlPrefix,
		conf:            conf,
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
	b, err := NodeInfoWithRequest(api.localNode, r)
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

	b, err := NodeInfoWithRequest(api.localNode, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.network.MessageBroker().Response(w, b)
}

var HandleTransactionCheckerFuncs = []common.CheckerFunc{
	TransactionUnmarshal,
	HasTransaction,
	SaveTransactionHistory,
	MessageHasSameSource,
	MessageValidate,
	PushIntoTransactionPoolFromClient,
	BroadcastTransaction,
}

var HandleTransactionCheckerFuncsWithoutBroadcast = []common.CheckerFunc{
	TransactionUnmarshal,
	HasTransaction,
	SaveTransactionHistory,
	MessageHasSameSource,
	MessageValidate,
	PushIntoTransactionPoolFromNode,
}

func (api NetworkHandlerNode) ReceiveTransaction(body []byte, funcs []common.CheckerFunc) (transaction.Transaction, error) {
	message := common.NetworkMessage{Type: common.TransactionMessage, Data: body}
	checker := &MessageChecker{
		DefaultChecker:  common.DefaultChecker{Funcs: funcs},
		Consensus:       api.consensus,
		TransactionPool: api.transactionPool,
		Storage:         api.storage,
		LocalNode:       api.localNode,
		NetworkID:       api.consensus.NetworkID,
		Message:         message,
		Log:             log,
		Conf:            api.conf,
	}

	err := common.RunChecker(checker, common.DefaultDeferFunc)
	// Unfreezing payment can be sent repeatedly before expiration of unfreezing period.
	// Transaction history of unfreezing payment should not saved during that period.
	if err != nil {
		if len(checker.Transaction.H.Hash) > 0 && err != errors.UnfreezingNotReachedExpiration {
			block.SaveTransactionHistory(api.storage, checker.Transaction, block.TransactionHistoryStatusRejected)
		}
		return transaction.Transaction{}, err
	}

	return checker.Transaction, nil
}

func (api NetworkHandlerNode) MessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	if _, err = api.ReceiveTransaction(body, HandleTransactionCheckerFuncsWithoutBroadcast); err != nil {
		http.Error(w, err.Error(), httputils.StatusCode(err))
		return
	}
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

func NodeInfoWithRequest(localNode *node.LocalNode, r *http.Request) (b []byte, err error) {
	var endpoint string
	if localNode.PublishEndpoint() != nil {
		endpoint = localNode.PublishEndpoint().String()
	} else {
		rUrl := common.RequestURLFromRequest(r)
		rUrl.Path = ""
		rUrl.RawQuery = ""
		endpoint = rUrl.String()
	}

	info := map[string]interface{}{
		"address":    localNode.Address(),
		"alias":      localNode.Alias(),
		"endpoint":   endpoint,
		"state":      localNode.State().String(),
		"validators": localNode.GetValidators(),
	}

	b, err = json.Marshal(info)
	return
}
