package core

import (
	"boscoin.io/sebak/pkg/core/dbi"
	"boscoin.io/sebak/pkg/core/types"
	"boscoin.io/sebak/pkg/rawdb"
	"sync"
)

type BlockChain struct {
	lock sync.Mutex

	metaDb  *dbi.MetaDb
	chainDb *dbi.ChainDb

	state        *State
	genesisBlock *types.Block
	currentBlock *types.Block

	// cache
	chainState *types.ChainState
}

func NewBlockChain(raw rawdb.Database) *BlockChain {
	b := &BlockChain{
		chainDb: dbi.NewChainDb(raw, raw),
		metaDb:  dbi.NewMetaDb(raw, raw),
		genesisBlock: types.NewBlock(&types.BlockHeader{
			Version:        uint32(1),
			PrevHeaderHash: types.Uint256{},
			MerkleRoot:     types.Uint256{},
			Timestamp:      uint64(0),
		}, []*types.Transaction{}),
	}

	b.initState()

	return b
}

func (o *BlockChain) CommitBlock(block *types.Block) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.currentBlock = block
}

func (o *BlockChain) State() *types.ChainState {
	return o.chainState
}

func (o *BlockChain) initState() {
	initialized := o.metaDb.HasChainState()

	if !initialized {
		o.chainDb.WriteBlock(0, o.genesisBlock.Header.Hash(), o.genesisBlock)
		o.chainDb.WriteHeader(0, o.genesisBlock.Header)

		o.chainState = &types.ChainState{
			Hash:              o.genesisBlock.Header.Hash(),
			Height:            0,
			NumTransactions:   0,
			TotalTransactions: 0,
		}
		o.metaDb.WriteChainState(o.chainState)
	}
}
