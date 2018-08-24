package statedb

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/trie"
	"bytes"
	"fmt"
)

type Storage map[sebakcommon.Hash][]byte

type Code []byte

var emptyCodeHash = sebakcommon.MakeHash([]byte{})

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

func (so *stateObject) Balance() string {
	return so.data.Balance
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

func (so *stateObject) Checkpoint() string {
	return so.data.Checkpoint
}

func (so *stateObject) GetState(key sebakcommon.Hash) []byte {
	value, exists := so.cachedStorage[key]
	if exists {
		return value
	}
	value, err := so.storageTrie.TryGet(key[:])
	if err != nil {
		return []byte{}
	}
	if bytes.Compare(value, []byte{}) == 0 {
		so.cachedStorage[key] = value
	}
	return value
}

/* SETTERS */
func (so *stateObject) SetState(key sebakcommon.Hash, value []byte) {
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

func (so *stateObject) AddBalanceWithCheckpoint(amount sebakcommon.Amount, checkpoint string) (err error) {
	so.data.Checkpoint = checkpoint
	so.AddBalance(amount)
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

func (so *stateObject) SubBalanceWithCheckpoint(amount sebakcommon.Amount, checkpoint string) (err error) {
	so.data.Checkpoint = checkpoint
	so.SubBalance(amount)
	return
}

func (so *stateObject) SetCheckpoint(checkpoint string) {
	so.data.Checkpoint = checkpoint
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
		if bytes.Compare(value, []byte{}) == 0 {
			continue
		}
		so.storageTrie.TryUpdate(key[:], value[:])
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

func (so *stateObject) CommitDB(root sebakcommon.Hash) (err error) {

	if so.dirtyCode == true {
		so.db.Put(so.data.CodeHash, so.code)
		so.dirtyCode = false
	}

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
		createdKey := block.GetBlockAccountCreatedKey(sebakcommon.GetUniqueIDFromUUID())
		err = st.New(createdKey, so.Address())
	}
	if err == nil {
		event := "saved"
		event += " " + fmt.Sprintf("address-%s", so.Address())
		observer.BlockAccountObserver.Trigger(event, &so.data)
	}

	bac := block.BlockAccountCheckpoint{
		Checkpoint: so.data.Checkpoint,
		Address:    so.Address(),
		Balance:    so.data.Balance,
	}
	err = bac.Save(st)

	return
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
