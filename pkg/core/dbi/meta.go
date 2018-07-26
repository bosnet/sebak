package dbi

import (
	"boscoin.io/sebak/pkg/core/types"
)

var (
	chainState = []byte("#ChainState")
)

type MetaDb struct {
	reader DatabaseReader
	writer DatabaseWriter
}

func NewMetaDb(reader DatabaseReader, writer DatabaseWriter) *MetaDb {
	return &MetaDb{
		reader: reader,
		writer: writer,
	}
}

func (o *MetaDb) WriteChainState(state *types.ChainState) {
	data, err := types.Serialize(state)
	if err != nil {
		panic(err)
	}
	if err := o.writer.Put(chainState, data); err != nil {
		panic(err)
	}
}

func (o *MetaDb) HasChainState() bool {
	if has, err := o.reader.Has(chainState); err != nil {
		panic(err)
	} else {
		return has
	}
}

func (o *MetaDb) ReadChainState() *types.ChainState {
	state := &types.ChainState{}
	if data, err := o.reader.Get(chainState); err != nil {
		panic(err)
	} else if err := types.Deserialize(data, state); err != nil {
		panic(err)
	}

	return state
}
