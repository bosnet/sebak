package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

type HelperTestGetBlocksHandler struct {
	st     *storage.LevelDBBackend
	server *httptest.Server
	blocks []block.Block
}

func (p *HelperTestGetBlocksHandler) Prepare() {
	p.st, _ = storage.NewTestMemoryLevelDBBackend()

	apiHandler := NetworkHandlerNode{storage: p.st}

	router := mux.NewRouter()
	router.HandleFunc(GetBlocksPattern, apiHandler.GetBlocksHandler).Methods("GET")

	p.server = httptest.NewServer(router)

	var bks []block.Block
	for i := 0; i < 5; i++ {
		bks = append(bks, p.createBlock())
	}

	p.blocks = bks

	return
}

func (p *HelperTestGetBlocksHandler) createBlock() block.Block {
	var txs []transaction.Transaction
	var txHashes []string
	for j := 0; j < 4; j++ {
		_, tx := transaction.TestMakeTransaction(networkID, 3)
		txHashes = append(txHashes, tx.GetHash())
		txs = append(txs, tx)
	}

	var height int
	latest, err := block.GetLatestBlock(p.st)
	if err == nil {
		height = int(latest.Height)
	} else {
		if _, ok := err.(*errors.Error); !ok {
			panic(err)
		}
		height = -1
	}
	bk := block.TestMakeNewBlock(txHashes)
	bk.Height = uint64(height + 1)
	bk.Save(p.st)

	for _, tx := range txs {
		b, _ := tx.Serialize()
		btx := block.NewBlockTransactionFromTransaction(bk.Hash, bk.Height, tx, b)
		if err = btx.Save(p.st); err != nil {
			panic(err)
		}
	}

	return bk
}

func (p *HelperTestGetBlocksHandler) URL(urlValues url.Values) (u *url.URL) {
	u, _ = url.Parse(p.server.URL)
	u.Path = GetBlocksPattern

	if urlValues != nil {
		u.RawQuery = urlValues.Encode()
	}

	return
}

func (p *HelperTestGetBlocksHandler) Done() {
	p.server.Close()
	p.st.Close()
}

func (p *HelperTestGetBlocksHandler) UnmarshalFromResponseBody(body io.ReadCloser) (
	rbs map[GetBlocksDataType][]interface{},
	err error,
) {
	defer body.Close()

	rbs = map[GetBlocksDataType][]interface{}{}

	sc := bufio.NewScanner(body)
	for sc.Scan() {
		var itemType GetBlocksDataType
		var b interface{}
		itemType, b, err = UnmarshalGetBlocksHandlerItem(sc.Bytes())

		rbs[itemType] = append(rbs[itemType], b)
	}
	if err = sc.Err(); err != nil {
		return
	}

	return
}

// TestGetBlocksHandler will check `/blocks` api returns the correct `Block`
// list.
func TestGetBlocksHandler(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	u := p.URL(nil)

	req, err := http.NewRequest("GET", u.String(), nil)
	require.Nil(t, err)
	resp, err := p.server.Client().Do(req)
	require.Nil(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	rbs, err := p.UnmarshalFromResponseBody(resp.Body)
	require.Nil(t, err)
	require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeHeader]))

	for i, b := range p.blocks {
		rb := rbs[GetBlocksDataTypeHeader][i].(block.Header)
		require.Equal(t, b.Height, rb.Height)

		s, _ := b.Header.Serialize()
		rs, _ := rb.Serialize()
		require.Equal(t, s, rs)
	}
}

// TestGetBlocksHandlerOptions will check `/blocks` api returns the
// correct `Block` list with `GetBlocksOptions`
func TestGetBlocksHandlerOptions(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // empty options
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetMode(GetBlocksOptionsModeBlock)
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeBlock][i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // options.Limit = 3
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetMode(GetBlocksOptionsModeBlock).SetLimit(3)
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, int(options.Limit()), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range p.blocks[:options.Limit()] {
			rb := rbs[GetBlocksDataTypeBlock][i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // options.Reverse = true
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetMode(GetBlocksOptionsModeBlock).SetReverse(true)
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeBlock][len(p.blocks)-1-i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // options.Cursor set
		cursorIndex := 1
		expectedBlocks := p.blocks[cursorIndex+1:]

		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetMode(GetBlocksOptionsModeBlock).SetCursor([]byte(p.blocks[cursorIndex].Hash))
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(expectedBlocks), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range expectedBlocks {
			rb := rbs[GetBlocksDataTypeBlock][i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}
}

func TestGetBlocksHandlerWithInvalidLimit(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // options.Limit is string
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		urlValues := options.URLValues()
		urlValues.Set("limit", "killme")

		u := p.URL(urlValues)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}

	{ // options.Limit is negative
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		urlValues := options.URLValues()
		urlValues.Set("limit", "-100")

		u := p.URL(urlValues)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}
}

func TestGetBlocksHandlerWithInvalidReverse(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // options.Reverse unknown value
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		urlValues := options.URLValues()
		urlValues.Set("reverse", "killme")

		u := p.URL(urlValues)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}

	{ // options.Reverse capitalized
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		urlValues := options.URLValues()
		urlValues.Set("reverse", "TRUE")

		u := p.URL(urlValues)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestGetBlocksHandlerWithUnknownCursor(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // options.Cursor unknown cursor
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetCursor([]byte("killme"))
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}
}

func TestGetBlocksHandlerWithHeightRange(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // set `height-range`
		expectedLength := 2

		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetMode(GetBlocksOptionsModeBlock).SetHeightRange([2]uint64{p.blocks[1].Height, p.blocks[1+expectedLength].Height})
		u := p.URL(options.URLValues())

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, expectedLength, len(rbs[GetBlocksDataTypeBlock]))

		for i := 1; i < 1+expectedLength; i++ {
			b := p.blocks[i]
			rb := rbs[GetBlocksDataTypeBlock][i-1].(block.Block)
			require.Equal(t, b.Height, rb.Height)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}
}

func TestGetBlocksHandlerWithInvalidHeightRange(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // if value is missing, it will be ok
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		u := p.URL(options.URLValues())
		u.RawQuery = "height-range="

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeHeader]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeHeader][i].(block.Header)
			require.Equal(t, b.Height, rb.Height)

			s, _ := b.Header.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // wrong format
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		u := p.URL(options.URLValues())
		u.RawQuery = fmt.Sprintf("height-range=%d-", 1)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}

	{ // not uint64 value
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		u := p.URL(options.URLValues())
		u.RawQuery = fmt.Sprintf("height-range=%d-%s", 1, "findme")

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}

	{ // bigger start value than end
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		u := p.URL(options.URLValues())
		u.RawQuery = fmt.Sprintf("height-range=%d-%d", 1, 0)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)
	}

	{ // height is bigger than limit, set to limit
		var expectedLength uint64 = 2
		options, err := NewGetBlocksOptionsFromRequest(nil)
		require.Nil(t, err)
		options.SetHeightRange([2]uint64{0, p.blocks[len(p.blocks)-1].Height}).SetLimit(expectedLength)
		require.True(t, options.Height() > expectedLength)

		urlValues := options.URLValues()
		u := p.URL(urlValues)

		req, _ := http.NewRequest("GET", u.String(), nil)
		resp, _ := p.server.Client().Do(req)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, int(expectedLength), len(rbs[GetBlocksDataTypeHeader]))
	}
}

// TestGetBlocksHandlerWithModeBlock will check `/blocks` api returns the correct `Block`
// list with `mode` option.
func TestGetBlocksHandlerWithModeBlock(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{
		u := p.URL(nil)
		u.RawQuery = fmt.Sprintf("mode=%s", GetBlocksOptionsModeBlock)

		req, err := http.NewRequest("GET", u.String(), nil)
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeBlock][i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}
}

// TestGetBlocksHandlerWithModeHeader will check `/blocks` api returns the correct `Block`
// list with `mode` option.
func TestGetBlocksHandlerWithModeHeader(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // by default, mode will be `GetBlocksOptionsModeHeader`
		u := p.URL(nil)
		u.RawQuery = fmt.Sprintf("mode=%s", GetBlocksOptionsModeHeader)

		req, err := http.NewRequest("GET", u.String(), nil)
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeHeader]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeHeader][i].(block.Header)
			require.Equal(t, b.Height, rb.Height)

			s, _ := b.Header.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}
	}
}

func TestGetBlocksHandlerWithModeFull(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // by default, mode will be `GetBlocksOptionsModeFull`
		u := p.URL(nil)
		u.RawQuery = fmt.Sprintf("mode=%s", GetBlocksOptionsModeFull)

		req, err := http.NewRequest("GET", u.String(), nil)
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		rbs, err := p.UnmarshalFromResponseBody(resp.Body)
		require.Nil(t, err)
		require.Equal(t, len(p.blocks), len(rbs[GetBlocksDataTypeBlock]))

		for i, b := range p.blocks {
			rb := rbs[GetBlocksDataTypeBlock][i].(block.Block)
			require.Equal(t, b.Hash, rb.Hash)

			s, _ := b.Serialize()
			rs, _ := rb.Serialize()
			require.Equal(t, s, rs)
		}

		var expectedNumberOfTransactions int
		for _, b := range p.blocks {
			expectedNumberOfTransactions += len(b.Transactions)
		}
		require.Equal(t, expectedNumberOfTransactions, len(rbs[GetBlocksDataTypeTransaction]))

		var expectedNumberOfOperations int
		var txInxdex int
		var opInxdex int
		for _, b := range p.blocks {
			for _, txHash := range b.Transactions {
				tx, _ := block.GetBlockTransaction(p.st, txHash)
				expectedNumberOfOperations += len(tx.Operations)

				rtx := rbs[GetBlocksDataTypeTransaction][txInxdex].(block.BlockTransaction)
				require.Equal(t, tx.Hash, rtx.Hash)

				s, _ := tx.Serialize()
				rs, _ := rtx.Serialize()
				require.Equal(t, s, rs)

				for _, opHash := range tx.Operations {
					op, _ := block.GetBlockOperation(p.st, opHash)

					rop := rbs[GetBlocksDataTypeOperation][opInxdex].(block.BlockOperation)
					require.Equal(t, op.Hash, rop.Hash)

					s, _ := op.Serialize()
					rs, _ := rop.Serialize()
					require.Equal(t, s, rs)

					opInxdex++
				}

				txInxdex++
			}
		}
		require.Equal(t, expectedNumberOfOperations, len(rbs[GetBlocksDataTypeOperation]))
	}
}

// TestGetBlocksHandlerWithInvalidMode will check `/blocks` api returns error
// with invalid mode.
func TestGetBlocksHandlerWithInvalidMode(t *testing.T) {
	p := &HelperTestGetBlocksHandler{}
	p.Prepare()
	defer p.Done()

	{ // mode = 1
		u := p.URL(nil)
		u.RawQuery = "mode=1"

		req, err := http.NewRequest("GET", u.String(), nil)
		require.Nil(t, err)
		resp, err := p.server.Client().Do(req)
		require.Nil(t, err)

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		responseError := errors.Error{}
		err = json.Unmarshal(body, &responseError)
		require.Nil(t, err)

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, errors.ErrorInvalidQueryString.Code, responseError.Code)

	}
}
