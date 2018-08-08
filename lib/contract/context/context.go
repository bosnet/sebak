package context

import (
	sebak "boscoin.io/sebak/lib"

	"boscoin.io/sebak/lib/store/statestore"
)

type Context struct {
	SenderAccount *sebak.BlockAccount // Transaction.Source.
	//Transaction   *types.Transaction
	//BlockHeader   *types.BlockHeader

	StateStore *statestore.StateStore
	StateClone *statestore.StateClone

	//APICallCounter int // Simple version of PC
}
