package statedb

import (
	"boscoin.io/sebak/lib/block"
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

func New(root common.Hash, db *trie.EthDatabase) *StateDB {
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

func (stateDB *StateDB) ExistAccount(addr string) bool {
	if stateDB.getStateObject(addr) != nil {
		return true
	}
	return false
}

func (stateDB *StateDB) GetOrNewStateObject(addr string) *stateObject {
	stateObject := stateDB.getStateObject(addr)
	if stateObject == nil {
		stateObject = stateDB.createObject(addr)
	}
	return stateObject
}

func (stateDB *StateDB) GetCheckPoint(addr string) string {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Checkpoint()
	}
	return ""
}
func (stateDB *StateDB) GetBalance(addr string) string {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return "0"
}

func (stateDB *StateDB) GetCode(addr string) []byte {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (stateDB *StateDB) GetCodeHash(addr string) common.Hash {
	stateObject := stateDB.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (stateDB *StateDB) GetState(a string, b common.Hash) common.Hash {
	stateObject := stateDB.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b)
	}
	return common.Hash{}
}

func (stateDB *StateDB) CreateAccount(addr string) {
	stateDB.createObject(addr)
}

func (stateDB *StateDB) SetCheckpoint(addr string, checkpoint string) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCheckpoint(checkpoint)
	}
}

func (stateDB *StateDB) AddBalance(addr string, amount common.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (stateDB *StateDB) AddBalanceWithCheckpoint(addr string, amount common.Amount, checkpoint string) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalanceWithCheckpoint(amount, checkpoint)
	}
}

func (stateDB *StateDB) SubBalance(addr string, amount common.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (stateDB *StateDB) SubBalanceWithCheckpoint(addr string, amount common.Amount, checkpoint string) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalanceWithCheckpoint(amount, checkpoint)
	}
}

func (stateDB *StateDB) SetCode(addr string, code []byte) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(common.MakeHash(code), code)
	}
}

func (stateDB *StateDB) SetState(addr string, key, value common.Hash) {
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
	var data block.BlockAccount

	if err := data.Deserialize(enc); err != nil {
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
	newobj = newObject(addr, block.BlockAccount{Address: addr, Balance: "0"}, stateDB.db, stateDB.MarkStateObjectDirty)
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
	//TODO: store the data into db with prefixes

}

func (stateDB *StateDB) CommitTrie() (root common.Hash, err error) {

	for addr, stateObject := range stateDB.stateObjects {
		if _, isDirty := stateDB.stateObjectsDirty[addr]; isDirty {
			if _, err = stateObject.CommitTrie(); err != nil {
				return common.Hash{}, err
			}
			stateDB.updateStateObject(stateObject)
			delete(stateDB.stateObjectsDirty, addr)
			stateDB.stateObjectsCommitDirty[stateObject.Address()] = struct{}{}
		}
	}
	root, err = stateDB.trie.Commit(nil)
	return
}

func (stateDB *StateDB) CommitDB(root common.Hash) (err error) {
	for addr, stateObject := range stateDB.stateObjects {
		if _, isDirty := stateDB.stateObjectsCommitDirty[addr]; isDirty {
			if err = stateObject.CommitDB(stateObject.data.RootHash); err != nil {
				return
			}
			delete(stateDB.stateObjectsCommitDirty, addr)
		}
	}
	return stateDB.trie.CommitDB(root)
}
