package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/statedb/store"
	"encoding"
	"encoding/json"
	"fmt"
	dbm "github.com/tendermint/tendermint/libs/db"
)

type StateDB struct {
	db store.DB
	st store.Store

	stateObjects      map[string]*stateObject
	stateObjectsDirty map[string]struct{}
}

func New(db store.DB, root []byte, version int64) *StateDB {
	db = dbm.NewPrefixDB(db, []byte("s/k:"+"state"+"/"))
	st, err := store.New(db, store.PruneNothing, store.CommitID{Version: version, Hash: root})
	if err != nil {
		panic(err)
	}
	return &StateDB{
		db:                db,
		st:                st,
		stateObjects:      make(map[string]*stateObject),
		stateObjectsDirty: make(map[string]struct{}),
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

func (stateDB *StateDB) GetSequenceID(addr string) uint64 {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.SequenceID()
	}
	panic("No such sequenceID")
}
func (stateDB *StateDB) GetBalance(addr string) common.Amount {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return 0
}

func (stateDB *StateDB) GetCode(addr string) []byte {
	stateObject := stateDB.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (stateDB *StateDB) GetCodeHash(addr string) []byte {
	stateObject := stateDB.getStateObject(addr)
	if stateObject == nil {
		return []byte{}
	}
	return stateObject.CodeHash()
}

func (stateDB *StateDB) GetState(a string, b []byte) []byte {
	stateObject := stateDB.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(b)
	}
	return []byte{}
}

func (stateDB *StateDB) CreateAccount(addr string) {
	stateDB.createObject(addr)
}

func (stateDB *StateDB) SetSequenceID(addr string, sequenceID uint64) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetSequenceID(sequenceID)
	}
}

func (stateDB *StateDB) AddBalance(addr string, amount common.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (stateDB *StateDB) AddBalanceWithSequenceID(addr string, amount common.Amount, sequenceID uint64) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalanceWithSequenceID(amount, sequenceID)
	}
}

func (stateDB *StateDB) SubBalance(addr string, amount common.Amount) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (stateDB *StateDB) SubBalanceWithSequenceID(addr string, amount common.Amount, sequenceID uint64) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalanceWithSequenceID(amount, sequenceID)
	}
}

func (stateDB *StateDB) SetCode(addr string, code []byte) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(common.MakeHash(code), code)
	}
}

func (stateDB *StateDB) SetState(addr string, key, value []byte) {
	stateObject := stateDB.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

func (stateDB *StateDB) getStateObject(addr string) (stateObject *stateObject) {
	if obj := stateDB.stateObjects[addr]; obj != nil {
		return obj
	}
	enc := stateDB.st.Get([]byte(addr))

	if len(enc) == 0 {
		return nil
	}
	var data Account

	if err := deserialize(enc, &data); err != nil {
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
	newobj = newObject(addr, Account{Address: addr, Balance: 0}, stateDB.db, stateDB.MarkStateObjectDirty)
	stateDB.setStateObject(newobj)
	return newobj
}

func (stateDB *StateDB) updateStateObject(stateObject *stateObject) {
	addr := stateObject.Address()
	data, err := stateObject.Serialize()
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	stateDB.st.Set([]byte(addr), data)
	//TODO: store the data into db with prefixes

}

func (stateDB *StateDB) Commit() (commitID store.CommitID) {

	for addr, stateObject := range stateDB.stateObjects {
		if _, isDirty := stateDB.stateObjectsDirty[addr]; isDirty {
			stateObject.Commit()
			stateDB.updateStateObject(stateObject)
			delete(stateDB.stateObjectsDirty, addr)
		}
	}
	commitID = stateDB.st.Commit()
	return commitID
}

// Encapsulate serialization method for various functions
func serialize(i interface{}) ([]byte, error) {
	if bm, ok := i.(encoding.BinaryMarshaler); ok {
		return bm.MarshalBinary()
	}
	return json.Marshal(&i)
}

// Encapsulate deserialization method for various functions
func deserialize(data []byte, i interface{}) error {
	if bm, ok := i.(encoding.BinaryUnmarshaler); ok {
		return bm.UnmarshalBinary(data)
	}
	return json.Unmarshal(data, &i)
}
