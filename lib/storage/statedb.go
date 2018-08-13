package sebakstorage

import (
	"fmt"
	"reflect"
	"sort"

	"boscoin.io/sebak/lib/common"

	"github.com/btcsuite/btcutil/base58"
	"github.com/syndtr/goleveldb/leveldb"
)

type StateObjectState byte

const (
	StateObjectChanged StateObjectState = iota
	StateObjectDeleted
)

type StateObject struct {
	Key   string
	Value interface{}
	State StateObjectState
}

func NewStateObject(k string, i interface{}, s StateObjectState) *StateObject {
	obj := &StateObject{
		Key:   k,
		Value: i,
		State: s,
	}
	return obj
}

func (o *StateObject) MakeHash() ([]byte, error) {
	h, err := sebakcommon.MakeObjectHash(o)
	if err != nil {
		return nil, err
	}
	return h, err
}

func (o *StateObject) MakeHashString() (string, error) {
	h, err := o.MakeHash()
	if err != nil {
		return "", err
	}
	return base58.Encode(h), nil
}

type StateDB struct {
	levelDB *LevelDBBackend
	objects map[string]*StateObject
}

func NewStateDB(st *LevelDBBackend) *StateDB {
	db := &StateDB{
		levelDB: st,
		// If we need thread safety, we should use sync.Map insteads map
		objects: make(map[string]*StateObject),
	}
	return db
}

func (s *StateDB) Has(k string) (bool, error) {
	if _, ok := s.objects[k]; ok {
		return true, nil
	}
	return s.levelDB.Has(k)
}

func (s *StateDB) Get(k string, i interface{}) error {
	if obj, ok := s.objects[k]; ok {
		reflect.ValueOf(i).Elem().Set(reflect.ValueOf(obj.Value))
		return nil
	}

	return s.levelDB.Get(k, i)
}

func (s *StateDB) New(k string, i interface{}) error {
	s.objects[k] = NewStateObject(k, i, StateObjectChanged)
	return nil
}

func (s *StateDB) Set(k string, i interface{}) error {
	s.objects[k] = NewStateObject(k, i, StateObjectChanged)
	return nil
}

func (s *StateDB) Remove(k string) error {
	s.objects[k] = NewStateObject(k, nil, StateObjectDeleted)
	return nil
}

func (s *StateDB) GetIterator(prefix string, reverse bool) (func() (IterItem, bool), func()) {
	//TODO: support GetIterator
	panic("Not not support GetIterator in StateDB")
}

func (s *StateDB) News(vs ...Item) error {
	for _, v := range vs {
		s.New(v.Key, v.Value)
	}
	return nil
}

func (s *StateDB) Sets(vs ...Item) error {
	for _, v := range vs {
		s.Set(v.Key, v.Value)
	}
	return nil
}

func (s *StateDB) BatchWrite() error {
	batch := new(leveldb.Batch)
	for k, v := range s.objects {
		switch v.State {
		case StateObjectChanged:
			enc, err := s.levelDB.Encode(v.Value)
			if err != nil {
				return err
			}
			batch.Put(s.levelDB.makeKey(k), enc)
		case StateObjectDeleted:
			batch.Delete(s.levelDB.makeKey(k))
		}
	}

	if err := s.levelDB.core.Write(batch, nil); err != nil {
		return err
	}
	return nil
}

func (s *StateDB) Commit() error {
	return s.levelDB.Commit()
}

func (s *StateDB) Clean() error {
	s.objects = make(map[string]*StateObject)
	return nil
}

func (s *StateDB) Discard() error {
	return s.levelDB.Discard()
}

func (s *StateDB) MakeHash() ([]byte, error) {
	ks := make([]string, 0, len(s.objects))

	for k, _ := range s.objects {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	hashes := make([][]byte, 0, len(ks))
	for _, k := range ks {
		obj, ok := s.objects[k]
		if !ok {
			return nil, fmt.Errorf("Missing state key:%v", k)
		}
		h, err := obj.MakeHash()
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}

	h, err := sebakcommon.MakeObjectHash(hashes)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (s *StateDB) MakeHashString() (string, error) {
	h, err := s.MakeHash()
	if err != nil {
		return "", err
	}

	return base58.Encode(h), nil
}
