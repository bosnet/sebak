package types

type ChainState struct {
	Hash              Uint256
	Height            uint64
	NumTransactions   uint64
	TotalTransactions uint64
}
