package store

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/store"
)

type iavlStore interface {
	CommitKVStore
	Queryable
	VersionExists(version int64) bool
}

const (
	AccountStoreKeyString = "ac"
)

type Store interface {
	Set(key, value []byte)
	Get(key []byte) (value []byte)
	Has(key []byte) bool
	Hash() (hash []byte)
	LastCommitID() CommitID
	Commit() CommitID
	QueryForAccount(key []byte, height int64, getProve bool) ResponseQuery
}

type iavlStoreWrapper struct {
	st iavlStore
}

func New(db DB, strategy PruningStrategy, commitID CommitID) (Store, error) {

	st, err := store.LoadIAVLStore(db, commitID, strategy)
	if err != nil {
		return nil, err
	}
	iStore, ok := st.(iavlStore)
	if !ok && st.GetStoreType() != StoreTypeIAVL {
		return nil, fmt.Errorf("%v does not implement QueryableKVMultiStore", st)
	}
	return &iavlStoreWrapper{
		st: iStore,
	}, nil
}

func (s *iavlStoreWrapper) Set(key, value []byte) {
	s.st.Set(key, value)
}

func (s *iavlStoreWrapper) Get(key []byte) (value []byte) {
	return s.st.Get(key)
}

func (s *iavlStoreWrapper) Has(key []byte) bool {
	return s.st.Has(key)
}

func (s *iavlStoreWrapper) Hash() (hash []byte) {
	commitID := s.st.LastCommitID()
	return commitID.Hash
}

func (s *iavlStoreWrapper) LastCommitID() CommitID {
	return s.st.LastCommitID()
}

func (s *iavlStoreWrapper) Commit() CommitID {
	return s.st.Commit()
}

func (s *iavlStoreWrapper) Query(req RequestQuery) ResponseQuery {
	return s.st.Query(req)
}
func (s *iavlStoreWrapper) QueryForAccount(key []byte, height int64, getProve bool) ResponseQuery {
	req := RequestQuery{
		Data:   key,
		Path:   "/",
		Height: height,
		Prove:  getProve,
	}
	return s.Query(req)
}
