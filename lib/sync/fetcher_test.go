package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"

	"github.com/stretchr/testify/require"
)

func renderNodeItem(w http.ResponseWriter, itemType runner.NodeItemDataType, o interface{}) {
	s, err := json.Marshal(o)
	if err != nil {
		itemType = runner.NodeItemError
		s = []byte(err.Error())
	}

	writeNodeItem(w, itemType, s)
}

func writeNodeItem(w http.ResponseWriter, itemType runner.NodeItemDataType, s []byte) {
	w.Write(append([]byte(itemType+" "), append(s, '\n')...))
}

func TestBlockFetcher(t *testing.T) {
	st := block.InitTestBlockchain()
	defer st.Close()

	kp := keypair.Random()
	_, nw, localNode := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{
		allConnected: []string{kp.Address()},
		getNodeFunc: func(addr string) node.Node {
			ep, err := common.NewEndpointFromString("https://node1?NodeName=n1")
			require.NoError(t, err)
			v, err := node.NewValidator(kp.Address(), ep, "n1")
			require.NoError(t, err)
			return v
		},
	}

	bk := block.GetLatestBlock(st)
	bt, err := block.GetBlockTransaction(st, bk.Transactions[0])
	require.NoError(t, err)
	tp, err := block.GetTransactionPool(st, bt.Hash)
	require.NoError(t, err)
	bt.Message = tp.Message

	apiHandlerFunc := func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		renderNodeItem(w, runner.NodeItemBlock, bk)

		tp, _ := block.GetTransactionPool(st, bt.Hash)
		bt.Message = tp.Message

		renderNodeItem(w, runner.NodeItemBlockTransaction, bt)
		resp := w.Result()
		return resp, nil
	}

	f := NewBlockFetcher(nw, cm, st, localNode)
	f.apiClient = mockDoer{
		handleFunc: apiHandlerFunc,
	}
	//f.logger = log

	ctx := context.Background()
	si, err := f.Fetch(ctx, &SyncInfo{Height: 1})
	require.NoError(t, err)
	require.Equal(t, bk.Hash, si.Block.Hash)
	require.Equal(t, bk.TransactionsRoot, si.Block.TransactionsRoot)
}
