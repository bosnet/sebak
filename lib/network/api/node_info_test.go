package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/version"
)

func TestAPIGetNodeInfoHandler(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	endpoint, _ := common.ParseEndpoint("http://1.2.3.4:5678")
	kp, _ := keypair.Random()
	localNode, _ := node.NewLocalNode(kp, endpoint, "")

	nv := node.NodeVersion{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		GitState:  version.GitState,
		BuildDate: version.BuildDate,
	}

	nd := node.NodeInfoNode{
		Version:    nv,
		State:      localNode.State(),
		Alias:      localNode.Alias(),
		Address:    localNode.Address(),
		Endpoint:   nil,
		Validators: localNode.GetValidators(),
	}

	policy := node.NodePolicy{
		NetworkID:                 networkID,
		InitialBalance:            common.Amount(1000),
		BaseReserve:               common.BaseReserve,
		BaseFee:                   common.BaseFee,
		BlockTime:                 time.Duration(5) * time.Second,
		OperationsLimit:           1000,
		TransactionsLimit:         2000,
		GenesisBlockConfirmedTime: common.GenesisBlockConfirmedTime,
		InflationRatio:            common.InflationRatioString,
		BlockHeightEndOfInflation: common.BlockHeightEndOfInflation,
	}

	nodeInfo := node.NodeInfo{
		Node:   nd,
		Policy: policy,
	}

	latestBlock := block.Block{
		Header: block.Header{
			Height:   100,
			TotalTxs: 9,
		},
		Hash: "findme",
	}
	apiHandler := NetworkHandlerAPI{localNode: localNode, storage: st, nodeInfo: nodeInfo}
	apiHandler.GetLatestBlock = func() block.Block {
		return latestBlock
	}

	router := mux.NewRouter()
	router.HandleFunc(GetNodeInfoPattern, apiHandler.GetNodeInfoHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	body, err := request(ts, GetNodeInfoPattern, false)
	require.Nil(t, err)
	data, err := ioutil.ReadAll(bufio.NewReader(body))
	body.Close()

	require.NotEmpty(t, data)

	receivedNodeInfo, err := node.NewNodeInfoFromJSON(data)
	require.Nil(t, err)

	require.NotNil(t, receivedNodeInfo.Node.Endpoint)

	// if `node.NodeInfo.Node.Endpoint` is nil, the server URL must be
	// `Endpoint` in the response body.
	require.Equal(t, ts.URL, receivedNodeInfo.Node.Endpoint.String())
	require.Equal(t, len(nodeInfo.Node.Validators), len(receivedNodeInfo.Node.Validators))
	require.Equal(t, latestBlock.Height, receivedNodeInfo.Block.Height)
	require.Equal(t, latestBlock.Hash, receivedNodeInfo.Block.Hash)
	require.Equal(t, latestBlock.TotalTxs, receivedNodeInfo.Block.TotalTxs)

	js, _ := json.Marshal(policy)
	rjs, _ := json.Marshal(receivedNodeInfo.Policy)
	require.Equal(t, js, rjs)

	// udpate localNode state
	localNode.SetBooting()

	body, err = request(ts, GetNodeInfoPattern, false)
	require.Nil(t, err)
	data, err = ioutil.ReadAll(bufio.NewReader(body))
	body.Close()

	receivedNodeInfo, _ = node.NewNodeInfoFromJSON(data)
	require.Equal(t, localNode.State(), receivedNodeInfo.Node.State)
}
