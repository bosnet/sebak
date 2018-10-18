package api

import (
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

func (api NetworkHandlerAPI) GetNodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	nodeInfo := &node.NodeInfo{}
	*nodeInfo = *&api.nodeInfo

	nodeInfo.Node.State = api.localNode.State()

	if nodeInfo.Node.Endpoint == nil {
		rUrl := common.RequestURLFromRequest(r)
		rUrl.Path = ""
		rUrl.RawQuery = ""

		nodeInfo.Node.Endpoint = common.NewEndpointFromURL(rUrl)
	}

	if api.GetLatestBlock != nil {
		latestBlock := api.GetLatestBlock()
		nodeInfo.Block = node.NodeBlockInfo{
			Height:   latestBlock.Height,
			Hash:     latestBlock.Hash,
			TotalTxs: latestBlock.TotalTxs,
		}
	}

	var b []byte
	var err error
	if b, err = common.JSONMarshalIndent(nodeInfo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(b)
}
