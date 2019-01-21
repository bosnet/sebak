package store

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/store"
)

type QueryableCommitKVStore interface {
	CommitKVStore
	Queryable
}

type IAVLStore struct {
	st QueryableCommitKVStore
}

func NewIAVLStore(db DB, strategy PruningStrategy) (*IAVLStore, error) {
	iavl, err := store.LoadIAVLStore(db, CommitID{}, strategy)
	if err != nil {
		return nil, err
	}
	queryableCommitKVStore, ok := iavl.(QueryableCommitKVStore)
	if !ok {
		return nil, fmt.Errorf("iavlStore is not satisfy the interface")
	}

	return &IAVLStore{
		st: queryableCommitKVStore,
	}, nil
}

func (s *IAVLStore) Get(key []byte) []byte {
	return s.st.Get(key)
}

func (s *IAVLStore) Has(key []byte) bool {
	return s.st.Has(key)
}

func (s *IAVLStore) Set(key, value []byte) {
	s.st.Set(key, value)
}

func (s *IAVLStore) Delete(key []byte) {
	s.st.Delete(key)
}

func (s *IAVLStore) Iterator(start, end []byte) Iterator {
	return s.st.Iterator(start, end)
}

func (s *IAVLStore) ReverseIterator(start, end []byte) Iterator {
	return s.st.ReverseIterator(start, end)
}

func (s *IAVLStore) SetPruning(pruning PruningStrategy) {
	s.st.SetPruning(pruning)
}

func (s *IAVLStore) LastCommitID() CommitID {
	return s.st.LastCommitID()
}

func (s *IAVLStore) Commit() CommitID {
	return s.st.Commit()
}

func (s *IAVLStore) Query(req RequestQuery) ResponseQuery {
	return s.st.Query(req)
}
