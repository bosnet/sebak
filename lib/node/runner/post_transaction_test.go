package runner

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner/api"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

func TestPostTransaction(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	endpoint, _ := common.NewEndpointFromString("http://localhost:12345")
	localNode, _ := node.NewLocalNode(keypair.Random(), endpoint, "")
	localNode.AddValidators(localNode.ConvertToValidator())
	isaac, _ := consensus.NewISAAC(
		localNode,
		nil,
		network.NewValidatorConnectionManager(localNode, nil, nil),
		st,
		common.NewTestConfig(),
		nil,
	)

	var config *network.HTTP2NetworkConfig

	config, _ = network.NewHTTP2NetworkConfigFromEndpoint(localNode.Alias(), endpoint)
	nt := network.NewHTTP2Network(config)

	nodeHandler := NetworkHandlerNode{
		storage:   st,
		consensus: isaac,
		network:   nt,
		localNode: localNode,
		conf:      common.Config{OpsLimit: 1},
	}
	apiHandler := api.NewNetworkHandlerAPI(localNode, nt, nil, "", node.NodeInfo{})

	router := mux.NewRouter()
	router.HandleFunc(
		api.GetTransactionsHandlerPattern,
		func(w http.ResponseWriter, r *http.Request) {
			apiHandler.PostTransactionsHandler(
				w, r,
				nodeHandler.ReceiveTransaction, HandleTransactionCheckerFuncs,
			)
			return
		},
	).Methods("POST")

	server := httptest.NewServer(router)
	defer server.Close()

	kp := keypair.Random()
	tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
	b, _ := tx.Serialize()

	{ // send broken json message
		req, _ := http.NewRequest("POST", server.URL+api.GetTransactionsHandlerPattern, bytes.NewReader(b[:10]))
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}
}
