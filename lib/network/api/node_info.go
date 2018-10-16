package api

import (
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

func (api NetworkHandlerAPI) GetNodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	nodeInfo := &node.NodeInfo{}
	*nodeInfo = *&api.nodeInfo

	if nodeInfo.Node.Endpoint == nil {
		rUrl := common.RequestURLFromRequest(r)
		rUrl.Path = ""
		rUrl.RawQuery = ""

		nodeInfo.Node.Endpoint = common.NewEndpointFromURL(rUrl)
	}

	if api.GetLatestBlock != nil {
		nodeInfo.Block = node.NodeBlockInfo{
			Height:   api.GetLatestBlock().Height,
			Hash:     api.GetLatestBlock().Hash,
			TotalTxs: api.GetLatestBlock().TotalTxs,
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
