package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/statedb/store"
	"bytes"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type Storage map[string][]byte

type Code []byte

var emptyCodeHash = common.MakeHash([]byte{})

type Account struct {
	Address    string        `json:"address"`
	Balance    common.Amount `json:"balance"`
	SequenceID uint64        `json:"sequence_id"`
	CodeHash   []byte        `json:"code_hash"`
	RootHash   []byte        `json:"root_hash"`
	Version    int64         `json:"version"`
}

type stateObject struct {
	address   string
	storageSt store.Store
	db        store.DB
	data      Account
	code      Code

	cachedStorage Storage
	dirtyStorage  Storage

	dirtyCode bool
	onDirty   func(addr string)
}

func newObject(addr string, data Account, db store.DB, onDirty func(addr string)) *stateObject {
	//Prefix DB?
	db = dbm.NewPrefixDB(db, []byte("s/k:"+addr+"/"))
	st, err := store.New(db, store.PruneNothing, store.CommitID{Version: data.Version, Hash: data.RootHash})
	if err != nil {
		panic(err)
	}
	return &stateObject{
		address:       addr,
		storageSt:     st,
		db:            db,
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

func (so *stateObject) Serialize() ([]byte, error) {
	return serialize(so.data)
}

func (so *stateObject) Deserialize(encoded []byte) error {
	return deserialize(encoded, so.data)
}

/* GETTERS */
func (so *stateObject) Address() string {
	return so.address
}

func (so *stateObject) CodeHash() []byte {
	return so.data.CodeHash
}

func (so *stateObject) Balance() common.Amount {
	return so.data.Balance
}

func (so *stateObject) Code() []byte {
	if so.code != nil {
		return so.code
	} else if bytes.Equal(so.CodeHash(), emptyCodeHash) {
		return nil
	} else {
		code := so.db.Get(so.CodeHash())
		so.code = code
		return so.code
	}
}

func (so *stateObject) SequenceID() uint64 {
	return so.data.SequenceID
}

func (so *stateObject) GetState(key []byte) []byte {
	value, exists := so.cachedStorage[string(key)]
	if exists {
		return value
	}
	enc := so.storageSt.Get(key)
	if len(enc) > 0 {
		value = append(value, enc...)
	}
	so.cachedStorage[string(key)] = value
	return value
}

/* SETTERS */
func (so *stateObject) SetState(key, value []byte) {
	so.cachedStorage[string(key)] = value
	so.dirtyStorage[string(key)] = value

	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}

}

func (so *stateObject) AddBalance(amount common.Amount) error {
	val, err := so.Balance().Add(amount)
	if err != nil {
		return err
	}
	so.data.Balance = val
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return nil
}

func (so *stateObject) AddBalanceWithSequenceID(amount common.Amount, sequenceID uint64) error {
	if err := so.AddBalance(amount); err != nil {
		return err
	}
	so.data.SequenceID = sequenceID
	return nil
}

func (so *stateObject) SubBalance(amount common.Amount) error {
	val, err := so.Balance().Sub(amount)
	if err != nil {
		return err
	}
	so.data.Balance = val
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return nil
}

func (so *stateObject) SubBalanceWithSequenceID(amount common.Amount, sequenceID uint64) (err error) {
	if err := so.SubBalance(amount); err != nil {
		return err
	}
	so.data.SequenceID = sequenceID
	return nil
}

func (so *stateObject) SetSequenceID(sequenceID uint64) {
	so.data.SequenceID = sequenceID
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
}

func (so *stateObject) SetCode(codeHash, code []byte) {
	so.code = code
	so.data.CodeHash = codeHash
	so.dirtyCode = true
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
}

/* Trie Manipulation */
func (so *stateObject) updateStore() {
	for key, value := range so.dirtyStorage {
		delete(so.dirtyStorage, key)
		if len(value) == 0 {
			continue
		}
		so.storageSt.Set([]byte(key), value)
	}
}

func (so *stateObject) Commit() (commitID store.CommitID) {
	so.updateStore()
	commitID = so.storageSt.Commit()

	so.data.RootHash = commitID.Hash
	so.data.Version = commitID.Version

	return commitID
}
