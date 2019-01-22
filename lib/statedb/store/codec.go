package store

import (
	"github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
)

// Import cosmos-sdk/types/store.go for convenience.
// nolint
type (
	PruningStrategy = types.PruningStrategy
	CommitKVStore   = types.CommitKVStore
	CommitID        = types.CommitID
	Queryable       = types.Queryable
	RequestQuery    = abci.RequestQuery
	ResponseQuery   = abci.ResponseQuery
	DB              = dbm.DB
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
