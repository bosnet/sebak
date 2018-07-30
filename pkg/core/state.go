package core

import "boscoin.io/sebak/pkg/core/types"

type State struct {
	LastBlockHeight uint64

	LastBlockHash types.Uint256
}
