package store

import (
	"github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// Import cosmos-sdk/types/store.go for convenience.
// nolint
type (
	PruningStrategy  = types.PruningStrategy
	Store            = types.Store
	Committer        = types.Committer
	CommitStore      = types.CommitStore
	MultiStore       = types.MultiStore
	CacheMultiStore  = types.CacheMultiStore
	CommitMultiStore = types.CommitMultiStore
	KVStore          = types.KVStore
	KVPair           = types.KVPair
	Iterator         = types.Iterator
	CacheKVStore     = types.CacheKVStore
	CommitKVStore    = types.CommitKVStore
	CacheWrapper     = types.CacheWrapper
	CacheWrap        = types.CacheWrap
	CommitID         = types.CommitID
	StoreKey         = types.StoreKey
	StoreType        = types.StoreType
	Queryable        = types.Queryable
	TraceContext     = types.TraceContext
	Gas              = types.Gas
	GasMeter         = types.GasMeter
	GasConfig        = types.GasConfig
	RequestQuery     = abci.RequestQuery
	ResponseQuery    = abci.ResponseQuery
	DB               = dbm.DB
)

const (
	//nolint
	StoreTypeIAVL = types.StoreTypeIAVL
)

const (
	// PruneSyncable means only those states not needed for state syncing will be deleted (keeps last 100 + every 10000th)
	PruneSyncable = types.PruneSyncable

	// PruneEverything means all saved states will be deleted, storing only the current state
	PruneEverything = types.PruneEverything

	// PruneNothing means all historic states will be saved, nothing will be deleted
	PruneNothing = types.PruneNothing
)
