package dbi

import (
	"boscoin.io/sebak/pkg/rawdb"
	"testing"
)

func TestNewChainDb(t *testing.T) {
	rdb := rawdb.NewMemoryDb()
	NewChainDb(rdb, rdb)
}
