package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
	leveldbOpt "github.com/syndtr/goleveldb/leveldb/opt"

	"boscoin.io/sebak/lib/errors"
)

type Snapshot struct {
	*leveldb.Snapshot
}

func NewSnapshot(st *LevelDBBackend) (*Snapshot, error) {
	snapshot, err := st.DB.GetSnapshot()
	if err != nil {
		return nil, err
	}

	return &Snapshot{Snapshot: snapshot}, nil
}

func (s *Snapshot) Put([]byte, []byte, *leveldbOpt.WriteOptions) error {
	return errors.NotImplemented
}

func (s *Snapshot) Write(*leveldb.Batch, *leveldbOpt.WriteOptions) error {
	return errors.NotImplemented
}

func (s *Snapshot) Delete([]byte, *leveldbOpt.WriteOptions) error {
	return errors.NotImplemented
}
