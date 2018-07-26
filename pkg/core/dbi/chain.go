package dbi

import (
	"boscoin.io/sebak/pkg/core/types"
	"encoding/binary"
)

var (
	headerPrefix    = []byte("h")
	blockBodyPrefix = []byte("b")
)

type ChainDb struct {
	reader DatabaseReader
	writer DatabaseWriter
}

func NewChainDb(reader DatabaseReader, writer DatabaseWriter) *ChainDb {
	return &ChainDb{
		reader: reader,
		writer: writer,
	}
}

func (o *ChainDb) ReadHeader(height uint64) *types.BlockHeader {
	data, err := o.reader.Get(o.headerKey(height))
	if err != nil {
		panic(err)
	}

	msg := &types.BlockHeader{}
	types.Deserialize(data, msg)

	return msg
}

func (o *ChainDb) WriteHeader(height uint64, header *types.BlockHeader) {
	data, err := types.Serialize(header)
	if err != nil {
		panic(err)
	}
	if err := o.writer.Put(o.headerKey(height), data); err != nil {
		panic(err)
	}
}

func (o *ChainDb) WriteBlock(height uint64, hash types.Uint256, block *types.Block) {
	data, err := types.Serialize(&types.BlockBody{
		Transactions: block.Transactions,
	})
	if err != nil {
		panic(err)
	}
	if err := o.writer.Put(o.blockKey(height, hash), data); err != nil {
		panic(err)
	}
}

func (o *ChainDb) heightBytes(number uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, number)
	return b
}

func (o *ChainDb) headerKey(height uint64) []byte {
	return append(headerPrefix, o.heightBytes(height)...)
}

func (o *ChainDb) blockKey(height uint64, hash types.Uint256) []byte {
	return append(append(blockBodyPrefix, o.heightBytes(height)...), hash.Bytes()...)
}
