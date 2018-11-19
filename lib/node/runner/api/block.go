package api

import (
	"net/http"
	"strconv"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"

	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetBlockHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hashOrHeight"]
	if hash == "" {
		err := errors.BadRequestParameter
		httputils.WriteJSONError(w, err)
		return
	}

	var isHash bool
	height, err := strconv.ParseUint(hash, 10, 64)
	if err != nil {
		isHash = true
	}

	var res resource.Resource
	{
		var b block.Block
		var err error
		if isHash {
			b, err = block.GetBlock(api.storage, hash)
		} else {
			b, err = block.GetBlockByHeight(api.storage, height)
		}

		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
		res = resource.NewBlock(&b)
	}
	httputils.MustWriteJSON(w, 200, res)
}
