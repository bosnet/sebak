package api

import (
	"net/http"
	"strconv"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
)

func (api NetworkHandlerAPI) GetBlocksHandler(w http.ResponseWriter, r *http.Request) {
	p, err := NewPageQuery(r, WithEncodePageCursor(false))
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var currHeight uint64
	var option *storage.WalkOption
	{
		var cursor string
		if p.Cursor() != nil {
			var err error
			currHeight, err = strconv.ParseUint(string(p.Cursor()), 10, 64)
			if err != nil {
				httputils.WriteJSONError(w, errors.BadRequestParameter)
				return
			}
			cursor = block.GetBlockKeyPrefixHeight(currHeight)
		}
		option = storage.NewWalkOption(cursor, p.Limit(), p.Reverse())
	}

	var (
		firstHeight uint64 = currHeight
		lastHeight  uint64 = currHeight
		blocks      []resource.Resource
	)
	{
		var cnt uint64 = 1
		err := block.WalkBlocks(api.storage, option, func(b *block.Block, key []byte) (next bool, err error) {
			if cnt == 1 {
				firstHeight = b.Height
			}
			lastHeight = b.Height
			blocks = append(blocks, resource.NewBlock(b))
			cnt++
			return true, nil
		})
		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
	}

	var (
		prevCursor []byte
		nextCursor []byte
	)
	if firstHeight <= 0 {
		firstHeight = 1
	}
	if lastHeight <= 0 {
		lastHeight = 1
	}
	switch p.Reverse() {
	case true:
		firstHeight = firstHeight + 1
		lastHeight = lastHeight - 1
	case false:
		firstHeight = firstHeight - 1
		lastHeight = lastHeight + 1
	}

	if firstHeight > 0 {
		prevCursor = []byte(strconv.FormatUint(firstHeight, 10))
	}
	if lastHeight > 0 {
		nextCursor = []byte(strconv.FormatUint(lastHeight, 10))
	}

	list := p.ResourceList(blocks, prevCursor, nextCursor)
	httputils.MustWriteJSON(w, 200, list)
}
