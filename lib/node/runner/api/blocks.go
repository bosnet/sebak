package api

import (
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
)

func (api NetworkHandlerAPI) GetBlocksHandler(w http.ResponseWriter, r *http.Request) {
	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var (
		cursor []byte
		blocks []resource.Resource
	)

	{
		option := p.WalkOption()
		if httputils.IsEventStream(r) {
			option.Limit = 10
		}
		err := block.WalkBlocks(api.storage, option, func(b *block.Block, key []byte) (next bool, err error) {
			blocks = append(blocks, resource.NewBlock(b))
			cursor = key
			return true, nil
		})
		if err != nil {
			httputils.WriteJSONError(w, err)
			return

		}
	}

	if httputils.IsEventStream(r) {
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		for _, b := range blocks {
			es.Render(b)
		}
		es.Run(observer.BlockObserver, block.EventBlockPrefix)
		return
	}

	list := p.ResourceList(blocks, cursor)
	httputils.MustWriteJSON(w, 200, list)
}
