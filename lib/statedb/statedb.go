package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/trie"
	"fmt"
)

type StateDB struct {
	db   *trie.EthDatabase
	trie *trie.Trie

	stateObjects            map[string]*stateObject
	stateObjectsDirty       map[string]struct{}
	stateObjectsCommitDirty map[string]struct{}
}

func New(root sebakcommon.Hash, db *trie.EthDatabase) *StateDB {
	return &StateDB{
		db:                      db,
		trie:                    trie.NewTrie(root, db),
		stateObjects:            make(map[string]*stateObject),
		stateObjectsDirty:       make(map[string]struct{}),
		stateObjectsCommitDirty: make(map[string]struct{}),
	}
}

func (stateDB *StateDB) MarkStateObjectDirty(addr string) {
	stateDB.stateObjectsDirty[addr] = struct{}{}
}

func (stateDB *StateDB) GetOrNewStateObject(addr string) *stateObject {
	stateObject := stateDB.getStateObject(addr)
	if stateObject == nil {
		stateObject = stateDB.createObject(addr)
	}
	return stateObject
}

func (stateDB *StateDB) GetCode(addr string) []byte {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (stateDB *StateDB) GetCodeHash(addr string) sebakcommon.Hash {
	stateObject := stateDB.getStateObject(addr)
	if stateObject == nil {
		return sebakcommon.Hash{}
	}
	return sebakcommon.BytesToHash(stateObject.CodeHash())
}

func (stateDB *StateDB) GetState(a string, b sebakcommon.Hash) sebakcommon.Hash {
	stateObject := stateDB.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b)
	}
	return sebakcommon.Hash{}
}

func (stateDB *StateDB) CreateAccount(addr string) {
	stateDB.createObject(addr)
}

func (stateDB *StateDB) AddBalance(addr string, amount sebakcommon.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (stateDB *StateDB) SubBalance(addr string, amount sebakcommon.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (stateDB *StateDB) SetCode(addr string, code []byte) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(sebakcommon.MakeHash(code), code)
	}
}

func (stateDB *StateDB) SetState(addr string, key, value sebakcommon.Hash) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

func (stateDB *StateDB) getStateObject(addr string) (stateObject *stateObject) {
	if obj := stateDB.stateObjects[addr]; obj != nil {
		return obj
	}
	enc, err := stateDB.trie.TryGet([]byte(addr))
	if err != nil {
		return nil
	}
	if len(enc) == 0 {
		return nil
	}
	var data Account
	if err := trie.DecodeBytes(enc, &data); err != nil {
		return nil
	}
	obj := newObject(addr, data, stateDB.db, stateDB.MarkStateObjectDirty)
	stateDB.setStateObject(obj)
	return obj
}

func (stateDB *StateDB) setStateObject(object *stateObject) {
	stateDB.stateObjects[object.Address()] = object
}

func (stateDB *StateDB) createObject(addr string) (newobj *stateObject) {
	newobj = newObject(addr, Account{}, stateDB.db, stateDB.MarkStateObjectDirty)
	stateDB.setStateObject(newobj)
	return newobj
}

func (stateDB *StateDB) updateStateObject(stateObject *stateObject) {
	addr := stateObject.Address()
	data, err := stateObject.Serialize()
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	stateDB.trie.TryUpdate([]byte(addr), data)
}

func (stateDB *StateDB) CommitTrie() (root sebakcommon.Hash, err error) {

	for addr, stateObject := range stateDB.stateObjects {
		if _, isDirty := stateDB.stateObjectsDirty[addr]; isDirty {
			if _, err = stateObject.CommitTrie(); err != nil {
				stateObject.CommitDB(stateObject.data.RootHash)
				return sebakcommon.Hash{}, err
			}
			stateDB.updateStateObject(stateObject)
			delete(stateDB.stateObjectsDirty, addr)
			stateDB.stateObjectsCommitDirty[stateObject.Address()] = struct{}{}
		}
	}
	root, err = stateDB.trie.Commit(nil)
	return
}

func (stateDB *StateDB) CommitDB(root sebakcommon.Hash) {
	for addr, stateObject := range stateDB.stateObjects {
		if _, isDirty := stateDB.stateObjectsCommitDirty[addr]; isDirty {
			stateObject.CommitDB(stateObject.data.RootHash)
			delete(stateDB.stateObjectsCommitDirty, addr)
		}
	}
	stateDB.trie.CommitDB(root)
}
