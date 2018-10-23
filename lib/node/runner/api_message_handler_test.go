package runner

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node/runner/api"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

type HelperTestNodeMessageHandler struct {
	HelperTestGetNodeTransactionsHandler

	conf        common.Config
	nodeHandler *NetworkHandlerNode
}

func (p *HelperTestNodeMessageHandler) Prepare() {
	p.HelperTestGetNodeTransactionsHandler.Prepare()

	p.conf = common.NewConfig()
	p.nodeHandler = NewNetworkHandlerNode(
		p.localNode,
		p.network,
		p.st,
		p.consensus,
		p.TransactionPool,
		network.UrlPathPrefixNode,
		p.conf,
	)

	// override existing handler
	p.router = mux.NewRouter()
	p.server = httptest.NewServer(p.router)
	p.router.HandleFunc(api.PostTransactionPattern, p.nodeHandler.MessageHandler).
		Methods("POST").
		MatcherFunc(common.PostAndJSONMatcher)
}

func (p *HelperTestNodeMessageHandler) URL(urlValues url.Values) (u *url.URL) {
	u, _ = url.Parse(p.server.URL)
	u.Path = api.PostTransactionPattern

	if urlValues != nil {
		u.RawQuery = urlValues.Encode()
	}

	return
}

func (p *HelperTestNodeMessageHandler) makeTransaction() (tx transaction.Transaction) {
	receiverKP, _ := keypair.Random()
	tx = transaction.MakeTransactionCreateAccount(p.genesisKeypair, receiverKP.Address(), common.BaseReserve)
	tx.B.SequenceID = p.genesisAccount.SequenceID
	tx.Sign(p.genesisKeypair, networkID)

	return
}

func TestNodeMessageHandler(t *testing.T) {
	p := &HelperTestNodeMessageHandler{}
	p.Prepare()
	defer p.Done()

	u := p.URL(nil)

	tx := p.makeTransaction()
	require.Nil(t, tx.IsWellFormed(networkID, p.conf))

	postData, _ := tx.Serialize()
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(postData))
	req.Header.Set("Content-Type", "application/json")
	require.Nil(t, err)
	resp, err := p.server.Client().Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, p.TransactionPool.Has(tx.GetHash()))
}

func TestNodeMessageHandlerNotWellformedTransaction(t *testing.T) {
	p := &HelperTestNodeMessageHandler{}
	p.Prepare()
	defer p.Done()

	u := p.URL(nil)

	{ // invalid signature
		tx := p.makeTransaction()
		tx.H.Signature = "findme"
		errIsWellformed := tx.IsWellFormed(networkID, p.conf)
		require.Equal(t, errors.ErrorInvalidTransaction.Code, errIsWellformed.(*errors.Error).Code)

		postData, _ := tx.Serialize()
		req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var responseError errors.Error
		{
			err := json.Unmarshal(body, &responseError)
			require.Nil(t, err)
		}
		require.Equal(t, responseError.Data["error"], errIsWellformed.(*errors.Error).Data["error"])
		require.Equal(
			t,
			responseError.Code,
			errIsWellformed.(*errors.Error).Code,
		)
	}

	{ // under BaseReserve
		tx := p.makeTransaction()
		tx.H.Signature = "findme"
		opb := tx.B.Operations[0].B.(operation.CreateAccount)
		opb.Amount = common.Amount(0)
		tx.B.Operations[0].B = opb
		tx.Sign(p.genesisKeypair, networkID)

		errIsWellformed := tx.IsWellFormed(networkID, p.conf)
		require.Equal(t, errors.ErrorOperationAmountUnderflow.Code, errIsWellformed.(*errors.Error).Code)

		postData, _ := tx.Serialize()
		req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var responseError errors.Error
		{
			err := json.Unmarshal(body, &responseError)
			require.Nil(t, err)
		}
		require.Equal(t, responseError.Data["error"], errIsWellformed.(*errors.Error).Data["error"])
		require.Equal(
			t,
			responseError.Code,
			errIsWellformed.(*errors.Error).Code,
		)

	}

	{ // already in history
		tx := p.makeTransaction()
		postData, _ := tx.Serialize()

		{
			req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(postData))
			req.Header.Set("Content-Type", "application/json")
			require.Nil(t, err)
			resp, err := p.server.Client().Do(req)
			require.Nil(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// send again
		req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(postData))
		req.Header.Set("Content-Type", "application/json")
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var responseError errors.Error
		{
			err := json.Unmarshal(body, &responseError)
			require.Nil(t, err)
		}
		require.Equal(t, responseError.Data["error"], errors.ErrorNewButKnownMessage.Data["error"])
		require.Equal(
			t,
			responseError.Code,
			errors.ErrorNewButKnownMessage.Code,
		)
	}
}
