package storage

import (
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"
	leveldbIterator "github.com/syndtr/goleveldb/leveldb/iterator"
	leveldbOpt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbUtil "github.com/syndtr/goleveldb/leveldb/util"

	"boscoin.io/sebak/lib/errors"
)

type LevelDBCore interface {
	Has([]byte, *leveldbOpt.ReadOptions) (bool, error)
	Get([]byte, *leveldbOpt.ReadOptions) ([]byte, error)
	NewIterator(*leveldbUtil.Range, *leveldbOpt.ReadOptions) leveldbIterator.Iterator
	Put([]byte, []byte, *leveldbOpt.WriteOptions) error
	Write(*leveldb.Batch, *leveldbOpt.WriteOptions) error
	Delete([]byte, *leveldbOpt.WriteOptions) error
}

type LevelDBBackend struct {
	DB *leveldb.DB

	Core LevelDBCore
}

func setLevelDBCoreError(err error) error {
	if err == nil {
		return nil
	}

	return errors.Newf(
		errors.StorageCoreError,
		"%s: %s", errors.StorageCoreError.Message, err.Error(),
	)
}

func (st *LevelDBBackend) Init(config *Config) (err error) {
	var db *leveldb.DB

	if config.Scheme == "file" {
		if db, err = leveldb.OpenFile(config.Path, nil); err != nil {
			err = setLevelDBCoreError(err)
			return
		}
	} else if config.Scheme == "memory" {
		sto := leveldbStorage.NewMemStorage()
		if db, err = leveldb.Open(sto, nil); err != nil {
			err = setLevelDBCoreError(err)
			return
		}
	}

	st.DB = db
	st.Core = db

	return
}

func (st *LevelDBBackend) Close() error {
	return st.DB.Close()
}

func (st *LevelDBBackend) OpenTransaction() (*LevelDBBackend, error) {
	_, ok := st.Core.(*leveldb.Transaction)
	if ok {
		return nil, errors.AlreadyCommittable
	}

	transaction, err := st.Core.(*leveldb.DB).OpenTransaction()
	if err != nil {
		err = setLevelDBCoreError(err)
		return nil, err
	}

	return &LevelDBBackend{
		DB:   st.DB,
		Core: transaction,
	}, nil
}

func (st *LevelDBBackend) OpenBatch() (*LevelDBBackend, error) {
	_, ok := st.Core.(*BatchCore)
	if ok {
		return nil, errors.AlreadyCommittable
	}

	return &LevelDBBackend{
		DB:   st.DB,
		Core: NewBatchCore(st.DB),
	}, nil
}

func (st *LevelDBBackend) Discard() error {
	var committable Committable
	var ok bool
	if committable, ok = st.Core.(Committable); !ok {
		return errors.NotCommittable
	}

	committable.Discard()

	return nil
}

func (st *LevelDBBackend) Commit() error {
	var committable Committable
	var ok bool
	if committable, ok = st.Core.(Committable); !ok {
		return errors.NotCommittable
	}

	return setLevelDBCoreError(committable.Commit())
}

func (st *LevelDBBackend) makeKey(key string) []byte {
	return []byte(key)
}

func (st *LevelDBBackend) Has(k string) (bool, error) {
	ok, err := st.Core.Has(st.makeKey(k), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, setLevelDBCoreError(err)
	}

	return ok, nil
}

func (st *LevelDBBackend) GetRaw(k string) (b []byte, err error) {
	var exists bool
	if exists, err = st.Has(k); err != nil || !exists {
		if !exists {
			err = errors.StorageRecordDoesNotExist
		}
		return
	}

	b, err = st.Core.Get(st.makeKey(k), nil)
	err = setLevelDBCoreError(err)

	return
}

func (st *LevelDBBackend) Get(k string, i interface{}) (err error) {
	var b []byte
	if b, err = st.GetRaw(k); err != nil {
		return
	}

	if err = json.Unmarshal(b, &i); err != nil {
		err = setLevelDBCoreError(err)
		return
	}

	return
}

func (st *LevelDBBackend) New(k string, v interface{}) (err error) {
	var exists bool
	if exists, err = st.Has(k); err != nil {
		return
	} else if exists {
		return errors.Newf(errors.StorageRecordAlreadyExists, "record {%v} already exists in storage", k)
	}

	var encoded []byte
	encoded, err = json.Marshal(v)
	if err != nil {
		return setLevelDBCoreError(err)
	}

	return setLevelDBCoreError(st.Core.Put(st.makeKey(k), encoded, nil))
}

func (st *LevelDBBackend) News(vs ...Item) (err error) {
	if len(vs) < 1 {
		err = setLevelDBCoreError(errors.New("empty values"))
		return
	}

	var exists bool
	for _, v := range vs {
		if exists, err = st.Has(v.Key); exists || err != nil {
			if exists {
				return errors.Newf(errors.StorageRecordAlreadyExists, "record {%v} already exists in storage", v.Key)
			}
			return
		}
	}

	batch := new(leveldb.Batch)
	for _, v := range vs {
		var encoded []byte
		if encoded, err = json.Marshal(v); err != nil {
			return setLevelDBCoreError(err)
		}

		batch.Put(st.makeKey(v.Key), encoded)
	}

	err = setLevelDBCoreError(st.Core.Write(batch, nil))

	return
}

func (st *LevelDBBackend) Set(k string, v interface{}) (err error) {
	var encoded []byte
	if encoded, err = json.Marshal(v); err != nil {
		return setLevelDBCoreError(err)
	}

	var exists bool
	if exists, err = st.Has(k); !exists || err != nil {
		if !exists {
			err = errors.StorageRecordDoesNotExist
			return
		}
		return
	}

	err = setLevelDBCoreError(st.Core.Put(st.makeKey(k), encoded, nil))

	return
}

func (st *LevelDBBackend) Sets(vs ...Item) (err error) {
	if len(vs) < 1 {
		err = setLevelDBCoreError(errors.New("empty values"))
		return
	}

	var exists bool
	for _, v := range vs {
		if exists, err = st.Has(v.Key); !exists || err != nil {
			if !exists {
				err = errors.StorageRecordDoesNotExist
				return
			}
			return
		}
	}

	batch := new(leveldb.Batch)
	for _, v := range vs {
		var encoded []byte
		if encoded, err = json.Marshal(v); err != nil {
			return setLevelDBCoreError(err)
		}

		batch.Put(st.makeKey(v.Key), encoded)
	}

	err = setLevelDBCoreError(st.Core.Write(batch, nil))

	return
}

func (st *LevelDBBackend) Remove(k string) (err error) {
	var exists bool
	if exists, err = st.Has(k); !exists || err != nil {
		if !exists {
			err = errors.StorageRecordDoesNotExist
			return
		}
		return
	}

	err = setLevelDBCoreError(st.Core.Delete(st.makeKey(k), nil))

	return
}

func (st *LevelDBBackend) GetIterator(prefix string, option ListOptions) (func() (IterItem, bool), func()) {
	var reverse = false
	var cursor []byte
	var limit uint64 = 0
	if option != nil {
		reverse = option.Reverse()
		cursor = option.Cursor()
		limit = option.Limit()
	}

	var dbRange *leveldbUtil.Range
	if len(prefix) > 0 {
		dbRange = leveldbUtil.BytesPrefix(st.makeKey(prefix))
	}

	iter := st.Core.NewIterator(dbRange, nil)

	var funcNext func() bool
	var seek func() bool

	if reverse {
		funcNext = iter.Prev
		if cursor == nil {
			seek = iter.Last
		} else {
			seek = func() bool {
				iter.Seek(cursor)
				return funcNext()
			}
		}
	} else {
		funcNext = iter.Next
		if cursor == nil {
			seek = iter.First
		} else {
			seek = func() bool {
				iter.Seek(cursor)
				return funcNext()
			}
		}
	}

	var n uint64 = 0
	return func() (IterItem, bool) {

			exists := false
			if n == 0 {
				exists = seek()
			} else {
				exists = funcNext()
			}

			if exists {
				n++
			}

			item := IterItem{N: n, Key: iter.Key(), Value: iter.Value()}

			if limit != 0 && n > limit {
				exists = false
			}

			return item, exists

		},
		func() {
			iter.Release()
		}
}

type (
	WalkFunc   func(key, value []byte) (bool, error)
	WalkOption struct {
		Cursor  string
		Limit   uint64
		Reverse bool
	}
)

func NewWalkOption(cursor string, limit uint64, reverse bool) *WalkOption {
	o := &WalkOption{
		Cursor:  cursor,
		Limit:   limit,
		Reverse: reverse,
	}
	return o
}

func (st *LevelDBBackend) Walk(prefix string, option *WalkOption, walkFunc WalkFunc) error {
	if option == nil {
		option = &WalkOption{
			Cursor:  prefix,
			Reverse: false,
			Limit:   10,
		}
	}

	var dbRange *leveldbUtil.Range
	if len(prefix) > 0 {
		dbRange = leveldbUtil.BytesPrefix(st.makeKey(prefix))
	}

	iter := st.Core.NewIterator(dbRange, nil)
	defer iter.Release()

	var iterFunc func() bool
	if option.Reverse == true {
		iterFunc = iter.Prev
	} else {
		iterFunc = iter.Next
	}

	var ok bool
	var cnt uint64 = 0

	cursor := option.Cursor
	reverse := option.Reverse
	if cursor == "" {
		cursor = prefix
		if reverse == true {
			ok = iter.Last()
		} else {
			ok = iter.First()
		}
	} else {
		ok = iter.Seek(st.makeKey(cursor))
	}

	for ; ok; ok = iterFunc() {
		if cnt >= option.Limit {
			return iter.Error()
		}

		if next, err := walkFunc(iter.Key(), iter.Value()); err != nil {
			return err
		} else if next == false {
			return iter.Error()
		}

		if iter.Error() != nil {
			return iter.Error()
		}
		cnt++
	}

	return iter.Error()
}
