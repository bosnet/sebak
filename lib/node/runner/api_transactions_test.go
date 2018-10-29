package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

type HelperTestGetNodeTransactionsHandler struct {
	localNode         *node.LocalNode
	st                *storage.LevelDBBackend
	server            *httptest.Server
	blocks            []block.Block
	transactionHashes []string
	consensus         *consensus.ISAAC
	TransactionPool   *transaction.Pool
	router            *mux.Router
	genesisAccount    *block.BlockAccount
	genesisKeypair    *keypair.Full
	network           *network.MemoryNetwork
}

func (p *HelperTestGetNodeTransactionsHandler) Prepare() {
	p.st = block.InitTestBlockchain()
	p.blocks = append(p.blocks, block.GetGenesis(p.st))

	kp, _ := keypair.Random()
	endpoint, _ := common.NewEndpointFromString(
		fmt.Sprintf("http://localhost:12345"),
	)
	p.localNode, _ = node.NewLocalNode(kp, endpoint, "")
	p.localNode.AddValidators(p.localNode.ConvertToValidator())

	_, p.network, _ = network.CreateMemoryNetwork(nil)
	p.network.SetLocalNode(p.localNode)

	isaac, _ := consensus.NewISAAC(
		networkID,
		p.localNode,
		nil,
		NewTestConnectionManager(p.localNode, nil, nil, make(chan struct{}, 100)),
		p.st,
		common.NewConfig(),
		nil,
	)
	p.consensus = isaac
	p.TransactionPool = transaction.NewPool()

	apiHandler := NetworkHandlerNode{storage: p.st, consensus: isaac, transactionPool: p.TransactionPool}

	p.router = mux.NewRouter()
	p.router.HandleFunc(GetTransactionPattern, apiHandler.GetNodeTransactionsHandler).Methods("GET", "POST")

	p.server = httptest.NewServer(p.router)

	p.genesisKeypair, _ = keypair.Random()
	p.genesisAccount = block.NewBlockAccount(p.genesisKeypair.Address(), common.MaximumBalance)
	p.genesisAccount.MustSave(p.st)

	for i := 0; i < 3; i++ {
		p.blocks = append(p.blocks, p.createBlock())
	}

	for j := 0; j < 3; j++ {
		_, tx := transaction.TestMakeTransaction(networkID, 2)
		p.TransactionPool.Add(tx)
	}

	return
}

func (p *HelperTestGetNodeTransactionsHandler) Done() {
	p.server.Close()
	p.st.Close()
}

func (p *HelperTestGetNodeTransactionsHandler) createBlock() block.Block {
	var txs []transaction.Transaction
	var txHashes []string
	for j := 0; j < 2; j++ {
		_, tx := transaction.TestMakeTransaction(networkID, 2)
		txHashes = append(txHashes, tx.GetHash())
		txs = append(txs, tx)
		p.transactionHashes = append(p.transactionHashes, tx.GetHash())
	}

	latest := block.GetLatestBlock(p.st)
	height := int(latest.Height)
	bk := block.TestMakeNewBlock(txHashes)
	bk.Height = uint64(height + 1)
	bk.MustSave(p.st)

	for _, tx := range txs {
		b, _ := tx.Serialize()
		btx := block.NewBlockTransactionFromTransaction(bk.Hash, bk.Height, bk.Confirmed, tx, b)
		if err := btx.Save(p.st); err != nil {
			panic(err)
		}
	}

	return bk
}

func (p *HelperTestGetNodeTransactionsHandler) URL(urlValues url.Values) (u *url.URL) {
	u, _ = url.Parse(p.server.URL)
	u.Path = GetTransactionPattern

	if urlValues != nil {
		u.RawQuery = urlValues.Encode()
	}

	return
}

func TestGetNodeTransactionsHandlerWithoutHashes(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	defer p.Done()

	u := p.URL(nil)

	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err)
	resp, err := p.server.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	responseError := errors.Error{}
	err = json.Unmarshal(body, &responseError)
	require.NoError(t, err)

	require.Equal(t, errors.InvalidQueryString.Code, responseError.Code)
}

func TestGetNodeTransactionsHandlerWithUnknownHashes(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	defer p.Done()

	{ // only unknown hash
		unknownHashKey := "unknown-hash-key"
		u := p.URL(nil)
		u.RawQuery = fmt.Sprintf("hash=%s", unknownHashKey)

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)
		require.Equal(t, 1, len(rbs[NodeItemError]))
		require.Equal(t, errors.TransactionNotFound.Code, rbs[NodeItemError][0].(*errors.Error).Code)
		require.Equal(t, unknownHashKey, rbs[NodeItemError][0].(*errors.Error).Data["hash"])
	}

	{ // unknown hash + known hash
		unknownHashKey := "unknown-hash-key"
		query := url.Values{"hash": []string{unknownHashKey, p.transactionHashes[0]}}
		u := p.URL(nil)
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)
		require.Equal(t, 1, len(rbs[NodeItemError]))
		require.Equal(t, errors.TransactionNotFound.Code, rbs[NodeItemError][0].(*errors.Error).Code)
		require.Equal(t, unknownHashKey, rbs[NodeItemError][0].(*errors.Error).Data["hash"])

		require.Equal(t, 1, len(rbs[NodeItemTransaction]))

		tx := rbs[NodeItemTransaction][0].(transaction.Transaction)
		require.Equal(t, p.transactionHashes[0], tx.GetHash())
	}
}

// TestGetNodeTransactionsHandlerPOST checks the basic response in POST method
func TestGetNodeTransactionsHandlerPOST(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	defer p.Done()

	{ // `Content-Type` must be `application/json`
		query := url.Values{"hash": []string{p.transactionHashes[0]}}
		u := p.URL(nil)
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("POST", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.NoError(t, err)

		require.Equal(t, errors.ContentTypeNotJSON.Code, responseError.Code)
	}

	{ // with `Content-Type=application/json`
		query := url.Values{"hash": []string{p.transactionHashes[1]}}
		u := p.URL(nil)
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("POST", u.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)

		require.Equal(t, 1, len(rbs))
		require.Equal(t, 1, len(rbs[NodeItemTransaction]))

		tx := rbs[NodeItemTransaction][0].(transaction.Transaction)
		require.Equal(t, p.transactionHashes[1], tx.GetHash())
	}
}

// TestGetNodeTransactionsHandlerWithMultipleHashes checks multiple transaction hash
func TestGetNodeTransactionsHandlerWithMultipleHashes(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	defer p.Done()

	{ // GET
		txHashes := []string{p.transactionHashes[1], p.transactionHashes[len(p.transactionHashes)-2]}

		u := p.URL(nil)
		query := url.Values{"hash": txHashes}
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)

		require.Equal(t, 1, len(rbs))
		require.Equal(t, len(txHashes), len(rbs[NodeItemTransaction]))

		for i, hash := range txHashes {
			tx := rbs[NodeItemTransaction][i].(transaction.Transaction)
			require.Equal(t, hash, tx.GetHash())
		}
	}

	{ // POST
		u := p.URL(nil)

		txHashes := []string{p.transactionHashes[1], p.transactionHashes[len(p.transactionHashes)-2]}
		var postData []string
		postData = append(postData, txHashes...)

		req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(common.MustJSONMarshal(postData)))
		req.Header.Set("Content-Type", "application/json")
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)

		require.Equal(t, 1, len(rbs))
		require.Equal(t, 2, len(rbs[NodeItemTransaction]))

		for i, hash := range txHashes {
			tx := rbs[NodeItemTransaction][i].(transaction.Transaction)
			require.Equal(t, hash, tx.GetHash())
		}
	}
}

// transactions in `Pool`
func TestGetNodeTransactionsHandlerInTransactionPool(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	defer p.Done()

	{
		var txHashes []string
		for key, _ := range p.TransactionPool.Pool {
			txHashes = append(txHashes, key)
			break // Only get the first value in the pool
		}

		u := p.URL(nil)
		query := url.Values{"hash": txHashes}
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
		require.NoError(t, err)

		require.Equal(t, 1, len(rbs))
		require.Equal(t, 1, len(rbs[NodeItemTransaction]))
		tx := rbs[NodeItemTransaction][0].(transaction.Transaction)
		require.Equal(t, txHashes[0], tx.GetHash())
	}
}

// TestGetNodeTransactionsHandlerTooManyHashes checks when the number of hashes
// reaches limit, `transaction.Limit`.
func TestGetNodeTransactionsHandlerTooManyHashes(t *testing.T) {
	p := &HelperTestGetNodeTransactionsHandler{}
	p.Prepare()
	p.consensus.Conf.TxsLimit = 2
	defer p.Done()

	{
		var txHashes []string
		for key, _ := range p.TransactionPool.Pool {
			txHashes = append(txHashes, key)
			break // Only get the first value in the pool
		}
		txHashes = append(txHashes, p.transactionHashes[0], p.transactionHashes[1], p.transactionHashes[2])

		query := url.Values{"hash": txHashes}
		u := p.URL(nil)
		u.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := p.server.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.NoError(t, err)

		require.Equal(t, errors.InvalidQueryString.Code, responseError.Code)
	}
}
