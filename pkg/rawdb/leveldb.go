package rawdb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

type LevelDb struct {
	db *leveldb.DB
}

func NewLevelDb(path string) (*LevelDb, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDb{
		db: db,
	}, nil
}

func (o *LevelDb) Put(key []byte, value []byte) error {
	return o.db.Put(key, value, nil)
}

func (o *LevelDb) Has(key []byte) (bool, error) {
	return o.db.Has(key, nil)
}

func (o *LevelDb) Get(key []byte) ([]byte, error) {
	dat, err := o.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (o *LevelDb) Delete(key []byte) error {
	return o.db.Delete(key, nil)
}

func (o *LevelDb) NewIterator() iterator.Iterator {
	return o.db.NewIterator(nil, nil)
}
