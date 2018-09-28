package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
	"github.com/stellar/go/keypair"
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

func TestBlockFullFetcher(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	kp, _ := keypair.Random()
	_, nw, _ := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{
		allConnected: []string{kp.Address()},
		getNodeFunc: func(addr string) node.Node {
			ep, err := common.NewEndpointFromString("https://node1?NodeName=n1")
			require.Nil(t, err)
			v, err := node.NewValidator(kp.Address(), ep, "n1")
			require.Nil(t, err)
			return v
		},
	}

	bk := block.TestMakeNewBlock([]string{})
	bk.Height = uint64(1)

	apiHandlerFunc := func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		renderNodeItem(w, runner.NodeItemBlock, bk)
		resp := w.Result()
		return resp, nil
	}

	f := NewBlockFullFetcher(nw, cm, st, func(f *BlockFullFetcher) {
		f.apiClient = mockDoer{
			handleFunc: apiHandlerFunc,
		}
	})
	defer f.Stop()

	input := make(chan *Message)
	err := f.Consume(input)
	require.Nil(t, err)

	output := f.Produce()

	msg := &Message{
		BlockHeight: uint64(1),
	}
	go func() {
		input <- msg
	}()

	fetchedMsg := <-output
	require.NotNil(t, fetchedMsg.Block)

}
