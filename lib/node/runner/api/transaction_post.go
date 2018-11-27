package api

import (
	"io"
	"io/ioutil"
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/transaction"
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

func (api NetworkHandlerAPI) PostTransactionsHandler(
	w http.ResponseWriter,
	r *http.Request,
	handler func([]byte, []common.CheckerFunc) (transaction.Transaction, error),
	funcs []common.CheckerFunc,
) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	if _, err = handler(body, funcs); err != nil {
		if _, ok := err.(*errors.Error); !ok {
			err = errors.HTTPProblem.Clone().SetData("error", err.Error())
		}
		httputils.WriteJSONError(w, err)
	} else {
		w.WriteHeader(200)
	}
}
