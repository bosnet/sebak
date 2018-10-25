package api

import (
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type TeeReadCloser struct {
	io.ReadCloser
	teeReader io.Reader
}

func (tee TeeReadCloser) Read(p []byte) (n int, err error) {
	return tee.teeReader.Read(p)
}

func NewTeeReadCloser(origin io.ReadCloser, w io.Writer) io.ReadCloser {
	return &TeeReadCloser{
		ReadCloser: origin,
		teeReader:  io.TeeReader(origin, w),
	}
}

func (api NetworkHandlerAPI) PostTransactionsHandler(w http.ResponseWriter, r *http.Request, handler http.HandlerFunc) {
	var bufferResponse bytes.Buffer
	writer := bufio.NewWriter(&bufferResponse)
	interceptedResponseWriter := httputils.NewResponseWriterInterceptor(w, writer)

	var bufferRequest bytes.Buffer
	r.Body = NewTeeReadCloser(r.Body, &bufferRequest)

	handler(interceptedResponseWriter, r)
	writer.Flush()

	if interceptedResponseWriter.StatusCode() != http.StatusOK {
		var errResponse errors.Error
		if err := json.Unmarshal(bufferResponse.Bytes(), &errResponse); err != nil {
			// Just bypass
			w.WriteHeader(interceptedResponseWriter.StatusCode())
			w.Write(bufferResponse.Bytes())
			return
		}
		httputils.WriteJSONError(w, &errResponse)
		return
	}

	var tx transaction.Transaction
	json.Unmarshal(bufferRequest.Bytes(), &tx)
	if err := httputils.WriteJSON(w, 200, resource.NewTransactionPost(tx)); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
