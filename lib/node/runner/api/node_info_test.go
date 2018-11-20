package api

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/version"
)

func TestAPIGetNodeInfoHandler(t *testing.T) {
	st := block.InitTestBlockchain()
	defer st.Close()

	endpoint, _ := common.ParseEndpoint("http://1.2.3.4:5678")
	kp := keypair.Random()
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
		NetworkID:                 string(networkID),
		InitialBalance:            common.Amount(1000),
		BaseReserve:               common.BaseReserve,
		BaseFee:                   common.BaseFee,
		BlockTime:                 time.Duration(5) * time.Second,
		OperationsLimit:           1000,
		TransactionsLimit:         2000,
		GenesisBlockConfirmedTime: common.GenesisBlockConfirmedTime,
		InflationRatio:            common.InflationRatioString,
		UnfreezingPeriod:          common.UnfreezingPeriod,
		BlockHeightEndOfInflation: common.BlockHeightEndOfInflation,
	}

	nodeInfo := node.NodeInfo{
		Node:   nd,
		Policy: policy,
	}

	apiHandler := NetworkHandlerAPI{
		localNode: localNode,
		storage:   st,
		nodeInfo:  nodeInfo,
		GetLatestBlock: func() block.Block {
			return block.GetLatestBlock(st)
		},
	}

	router := mux.NewRouter()
	router.HandleFunc(GetNodeInfoPattern, apiHandler.GetNodeInfoHandler).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	body := request(ts, GetNodeInfoPattern, false)
	data, err := ioutil.ReadAll(bufio.NewReader(body))
	body.Close()

	require.NotEmpty(t, data)

	receivedNodeInfo, err := node.NewNodeInfoFromJSON(data)
	require.NoError(t, err)

	require.NotNil(t, receivedNodeInfo.Node.Endpoint)

	// if `node.NodeInfo.Node.Endpoint` is nil, the server URL must be
	// `Endpoint` in the response body.
	latestBlock := block.GetLatestBlock(st)
	require.Equal(t, ts.URL, receivedNodeInfo.Node.Endpoint.String())
	require.Equal(t, len(nodeInfo.Node.Validators), len(receivedNodeInfo.Node.Validators))
	require.Equal(t, latestBlock.Height, receivedNodeInfo.Block.Height)
	require.Equal(t, latestBlock.Hash, receivedNodeInfo.Block.Hash)
	require.Equal(t, latestBlock.TotalTxs, receivedNodeInfo.Block.TotalTxs)
	require.Equal(t, latestBlock.TotalOps, receivedNodeInfo.Block.TotalOps)

	js, _ := json.Marshal(policy)
	rjs, _ := json.Marshal(receivedNodeInfo.Policy)
	require.Equal(t, js, rjs)

	// udpate localNode state
	localNode.SetBooting()

	body = request(ts, GetNodeInfoPattern, false)
	defer body.Close()
	data, err = ioutil.ReadAll(bufio.NewReader(body))

	receivedNodeInfo, _ = node.NewNodeInfoFromJSON(data)
	require.Equal(t, localNode.State(), receivedNodeInfo.Node.State)
}
