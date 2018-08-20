package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/trie"
	"bytes"
)

type Storage map[sebakcommon.Hash]sebakcommon.Hash

type Code []byte

var emptyCodeHash = sebakcommon.MakeHash([]byte{})

type Account struct {
	Balance  string
	CodeHash []byte
	RootHash sebakcommon.Hash
}

type stateObject struct {
	address     string
	storageTrie *trie.Trie
	db          *trie.EthDatabase
	data        Account
	code        Code

	cachedStorage Storage
	dirtyStorage  Storage

	dirtyCode bool
	onDirty   func(addr string)
}

func newObject(addr string, data Account, db *trie.EthDatabase, onDirty func(addr string)) *stateObject {

	return &stateObject{
		address:       addr,
		storageTrie:   trie.NewTrie(data.RootHash, db),
		db:            db,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

/* GETTERS */
func (so *stateObject) Serialize() ([]byte, error) {
	return trie.EncodeToBytes(so.data)
}

func (so *stateObject) Address() string {
	return so.address
}

func (so *stateObject) CodeHash() []byte {
	return so.data.CodeHash
}

func (so *stateObject) Balance() string {
	return so.data.Balance
}
func (so *stateObject) Code() []byte {
	if so.code != nil {
		return so.code
	} else if bytes.Equal(so.CodeHash(), emptyCodeHash) {
		return nil
	} else {
		return so.code
	}
}

func (so *stateObject) GetState(key sebakcommon.Hash) sebakcommon.Hash {
	value, exists := so.cachedStorage[key]
	if exists {
		return value
	}
	enc, err := so.storageTrie.TryGet(key[:])
	if err != nil {
		return sebakcommon.Hash{}
	}
	if len(enc) > 0 {
		content, err := trie.Split(enc)
		if err != nil {
		}
		value.SetBytes(content)
	}
	if (value != sebakcommon.Hash{}) {
		so.cachedStorage[key] = value
	}
	return value
}

/* SETTERS */
func (so *stateObject) SetState(key, value sebakcommon.Hash) {
	so.cachedStorage[key] = value
	so.dirtyStorage[key] = value

	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}

}

func (so *stateObject) AddBalance(amount sebakcommon.Amount) (err error) {
	val := sebakcommon.MustAmountFromString(so.Balance())
	val, err = val.Add(amount)
	so.data.Balance = val.String()
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return
}

func (so *stateObject) SubBalance(amount sebakcommon.Amount) (err error) {
	val := sebakcommon.MustAmountFromString(so.Balance())
	val, err = val.Sub(amount)
	so.data.Balance = val.String()
	if so.onDirty != nil {
		so.onDirty(so.Address())
		so.onDirty = nil
	}
	return
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
		if (value == sebakcommon.Hash{}) {
			continue
		}
		v, _ := trie.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		so.storageTrie.TryUpdate(key[:], v)
	}
}

func (so *stateObject) CommitTrie() (root sebakcommon.Hash, err error) {
	so.updateTrie()
	root, err = so.storageTrie.Commit(nil)
	if err == nil {
		so.data.RootHash = sebakcommon.Hash(root)
	}
	return
}

func (so *stateObject) CommitDB(root sebakcommon.Hash) {
	so.storageTrie.CommitDB(root)
}

/*
func (so *stateObject) GetValue(key string, value interface{}) (err error) {
	encKey, err := trie.EncodeToBytes(key)

	hashedKey := sebakcommon.MakeHash(encKey)
	hashedVal := so.GetState(sebakcommon.BytesToHash(hashedKey))
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

	hashedKey := sebakcommon.MakeHash(encKey)
	hashedVal := sebakcommon.MakeHash(encVal)
	err = so.db.Put(hashedVal[:], encVal)
	if err != nil {
		return
	}
	so.SetState(sebakcommon.BytesToHash(hashedKey), sebakcommon.BytesToHash(hashedVal))
	return
}
*/
