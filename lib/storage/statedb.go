package storage

import (
	"sort"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"

	"github.com/btcsuite/btcutil/base58"
)

type StateDB struct {
	levelDB     *LevelDBBackend
	changedkeys map[string]struct{}
}

func NewStateDB(st *LevelDBBackend) *StateDB {
	db := &StateDB{
		levelDB: st,
		// If we need thread safety, we should use sync.Map insteads map
		changedkeys: make(map[string]struct{}),
	}
	return db
}

func (s *StateDB) Has(k string) (bool, error) {
	return s.levelDB.Has(k)
}

func (s *StateDB) Get(k string, i interface{}) error {
	return s.levelDB.Get(k, i)
}

func (s *StateDB) New(k string, i interface{}) error {
	s.changedkeys[k] = struct{}{}
	return s.levelDB.New(k, i)
}

func (s *StateDB) Set(k string, i interface{}) error {
	s.changedkeys[k] = struct{}{}
	return s.levelDB.Set(k, i)
}

func (s *StateDB) Remove(k string) error {
	s.changedkeys[k] = struct{}{}
	return s.levelDB.Remove(k)
}

func (s *StateDB) GetIterator(prefix string, reverse bool) (func() (IterItem, bool), func()) {
	return s.levelDB.GetIterator(prefix, &DefaultListOptions{reverse: reverse})
}

func (s *StateDB) News(vs ...Item) error {
	for _, v := range vs {
		s.changedkeys[v.Key] = struct{}{}
	}

	return s.levelDB.News(vs...)
}

func (s *StateDB) Sets(vs ...Item) error {
	for _, v := range vs {
		s.changedkeys[v.Key] = struct{}{}
	}

	return s.levelDB.Sets(vs...)
}

func (s *StateDB) Commit() error {
	return s.levelDB.Commit()
}

func (s *StateDB) Discard() error {
	return s.levelDB.Discard()
}

func (s *StateDB) MakeHash() ([]byte, error) {
	ks := make([]string, 0, len(s.changedkeys))

	for k, _ := range s.changedkeys {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	hashes := make([][]byte, 0, len(ks))
	for _, k := range ks {
		bs, err := s.levelDB.GetRaw(k)
		if err == errors.ErrorStorageRecordDoesNotExist {
			bs = nil
		} else if err != nil {
			return nil, err
		}
		h := common.MakeHash(bs)
		hashes = append(hashes, h)
	}

	h, err := common.MakeObjectHash(hashes)
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
