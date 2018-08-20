package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func TestCreateAccount(statedb *StateDB) string {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := sebakcommon.Amount(2000)
	checkpoint := uuid.New().String()

	statedb.CreateAccount(address)
	statedb.AddBalanceWithCheckpoint(address, balance, checkpoint)
	return address
}

func TestNewStateDB(st *sebakstorage.LevelDBBackend) *StateDB {
	edb := trie.NewEthDatabase(st)
	return New(sebakcommon.Hash{}, edb)
}

func TestNewStateDBWithRoot(st *sebakstorage.LevelDBBackend, root sebakcommon.Hash) *StateDB {
	edb := trie.NewEthDatabase(st)
	return New(root, edb)
}
