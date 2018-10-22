package api

import (
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction"
	"bufio"
	"bytes"
	"encoding/json"
	logging "github.com/inconshreveable/log15"
	"io"
	"io/ioutil"
	"net/http"
)

var log logging.Logger = logging.New("module", "API")

type TeeReadCloser struct {
	io.ReadCloser
	teeReader io.Reader
}

func (tee TeeReadCloser) Read(p []byte) (n int, err error) {
	return tee.teeReader.Read(p)
}

func (api NetworkHandlerAPI) PostTransactionsHandler(w http.ResponseWriter, r *http.Request, handler http.HandlerFunc) {
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)
	r.Body = &TeeReadCloser{ReadCloser: r.Body, teeReader: tee}

	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	interceptedResponseWriter := httputils.NewResponseWriterInterceptor(w, writer)
	handler(interceptedResponseWriter, r)
	writer.Flush()

	if interceptedResponseWriter.StatusCode() != http.StatusOK {
		b, _ := ioutil.ReadAll(&buffer)
		var returned errors.Error
		json.Unmarshal(b, &returned)
		httputils.WriteJSONError(w, &returned)
		return
	}
	log.Error(buf.String())

	b, err := ioutil.ReadAll(tee)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
	var tx transaction.Transaction
	json.Unmarshal(b, &tx)
	rtp := resource.NewTransactionPost(tx)
	if err := httputils.WriteJSON(w, 200, rtp); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
