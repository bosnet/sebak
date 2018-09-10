package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage/block"
	"boscoin.io/sebak/lib/storage/statedb/trie"
	"bytes"
	"fmt"
)

type Storage map[common.Hash]common.Hash

type Code []byte

var emptyCodeHash = common.MakeHash([]byte{})

type stateObject struct {
	address     string
	storageTrie *trie.Trie
	db          *trie.EthDatabase
	data        block.BlockAccount
	code        Code

	cachedStorage Storage
	dirtyStorage  Storage

	dirtyCode bool
	onDirty   func(addr string)
}

func newObject(addr string, data block.BlockAccount, db *trie.EthDatabase, onDirty func(addr string)) *stateObject {

	return &stateObject{
		address:       addr,
		storageTrie:   trie.NewTrie(data.RootHash, db),
		db:            db,
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

func (so *stateObject) Serialize() ([]byte, error) {
	return so.data.Serialize()
}

func (so *stateObject) Deserialize(encoded []byte) error {
	return so.data.Deserialize(encoded)
}

/* GETTERS */
func (so *stateObject) Address() string {
	return so.address
}

func (so *stateObject) CodeHash() []byte {
	return so.data.CodeHash
}

func (so *stateObject) Balance() common.Amount {
	return so.data.GetBalance()
}

func (so *stateObject) Code() []byte {
	if so.code != nil {
		return so.code
	} else if bytes.Equal(so.CodeHash(), emptyCodeHash) {
		return nil
	} else {
		code, err := so.db.Get(so.CodeHash())
		if err != nil {
			return nil
		}
		so.code = code
		return so.code
	}
}

func (so *stateObject) SequenceID() uint64 {
	return so.data.SequenceID
}

func (so *stateObject) GetState(key common.Hash) common.Hash {
	value, exists := so.cachedStorage[key]
	if exists {
		return value
	}
	enc, err := so.storageTrie.TryGet(key[:])
	if err != nil {
		return common.Hash{}
	}
	if len(enc) > 0 {
		value.SetBytes(enc)
	}
	if (value != common.Hash{}) {
		so.cachedStorage[key] = value
	}
	return value
}

/* SETTERS */
func (so *stateObject) SetState(key, value common.Hash) {
	so.cachedStorage[key] = value
	so.dirtyStorage[key] = value

	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}

}

func (so *stateObject) AddBalance(amount common.Amount) (err error) {
	val, err := so.Balance().Add(amount)
	so.data.Balance = val
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return
}

func (so *stateObject) AddBalanceWithSequenceID(amount common.Amount, sequenceID uint64) (err error) {
	so.data.SequenceID = sequenceID
	so.AddBalance(amount)
	return
}

func (so *stateObject) SubBalance(amount common.Amount) (err error) {
	val, err := so.Balance().Sub(amount)
	so.data.Balance = val
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return
}

func (so *stateObject) SubBalanceWithSequenceID(amount common.Amount, sequenceID uint64) (err error) {
	so.data.SequenceID = sequenceID
	so.SubBalance(amount)
	return
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
func (so *stateObject) updateTrie() {
	for key, value := range so.dirtyStorage {
		delete(so.dirtyStorage, key)
		if (value == common.Hash{}) {
			continue
		}
		so.storageTrie.TryUpdate(key[:], value[:])
	}
}

func (so *stateObject) CommitTrie() (root common.Hash, err error) {
	so.updateTrie()
	root, err = so.storageTrie.Commit(nil)
	if err == nil {
		so.data.RootHash = common.Hash(root)
	}
	return
}

func (so *stateObject) CommitDB(root common.Hash) (err error) {
	if err = so.Save(); err != nil {
		return
	}
	if err = so.storageTrie.CommitDB(root); err != nil {
		return
	}
	return nil
}

func (so *stateObject) Save() (err error) {
	st := so.db.BackEnd()
	key := block.GetBlockAccountKey(so.Address())

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	}

	if exists {
		err = st.Set(key, so.data)
	} else {
		// TODO consider to use, [`Transaction`](https://godoc.org/github.com/syndtr/goleveldb/leveldb#DB.OpenTransaction)
		err = st.New(key, so.data)
		createdKey := block.GetBlockAccountCreatedKey(common.GetUniqueIDFromUUID())
		err = st.New(createdKey, so.Address())
	}
	if err == nil {
		event := "saved"
		event += " " + fmt.Sprintf("address-%s", so.Address())
		observer.BlockAccountObserver.Trigger(event, &so.data)
	}

	bac := block.BlockAccountSequenceID{
		SequenceID: so.data.SequenceID,
		Address:    so.Address(),
		Balance:    so.data.Balance,
	}
	err = bac.Save(st)

	return
}

/*
func (so *stateObject) GetValue(key string, value interface{}) (err error) {
	encKey, err := trie.EncodeToBytes(key)
	hashedKey := common.MakeHash(encKey)
	hashedVal := so.GetState(common.BytesToHash(hashedKey))
	encVal, err := so.db.Get(hashedVal[:])
	err = trie.DecodeBytes(encVal, value)
	return
}
func (so *stateObject) SetValue(key string, value interface{}) (err error) {
	encKey, err := trie.EncodeToBytes(key)
	if err != nil {
		return
	}
	encVal, err := trie.EncodeToBytes(value)
	if err != nil {
		return
	}
	hashedKey := common.MakeHash(encKey)
	hashedVal := common.MakeHash(encVal)
	err = so.db.Put(hashedVal[:], encVal)
	if err != nil {
		return
	}
	so.SetState(common.BytesToHash(hashedKey), common.BytesToHash(hashedVal))
	return
}
*/
