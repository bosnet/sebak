package storage

import (
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	leveldbIterator "github.com/syndtr/goleveldb/leveldb/iterator"
	leveldbOpt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbUtil "github.com/syndtr/goleveldb/leveldb/util"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
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

	return errors.NewError(
		errors.ErrorStorageCoreError.Code,
		fmt.Sprintf("%s: %s", errors.ErrorStorageCoreError.Message, err.Error()),
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
		return nil, errors.New("this is already *leveldb.Transaction")
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

func (st *LevelDBBackend) Discard() error {
	ts, ok := st.Core.(*leveldb.Transaction)
	if !ok {
		return setLevelDBCoreError(errors.New("this is not *leveldb.Transaction"))
	}

	ts.Discard()
	return nil
}

func (st *LevelDBBackend) Commit() error {
	ts, ok := st.Core.(*leveldb.Transaction)
	if !ok {
		return setLevelDBCoreError(errors.New("this is not *leveldb.Transaction"))
	}

	return setLevelDBCoreError(ts.Commit())
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
			err = errors.ErrorStorageRecordDoesNotExist
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
	var encoded []byte
	serializable, ok := v.(common.Serializable)
	if ok {
		encoded, err = serializable.Serialize()
	} else {
		encoded, err = common.EncodeJSONValue(v)
	}
	if err != nil {
		err = setLevelDBCoreError(err)
		return
	}

	var exists bool
	if exists, err = st.Has(k); exists || err != nil {
		if exists {
			err = errors.ErrorStorageRecordAlreadyExists
			return
		}
		return
	}

	err = setLevelDBCoreError(st.Core.Put(st.makeKey(k), encoded, nil))

	return
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
				err = errors.ErrorStorageRecordAlreadyExists
			}
			return
		}
	}

	batch := new(leveldb.Batch)
	for _, v := range vs {
		var encoded []byte
		if encoded, err = common.EncodeJSONValue(v); err != nil {
			err = setLevelDBCoreError(err)
			return
		}

		batch.Put(st.makeKey(v.Key), encoded)
	}

	err = setLevelDBCoreError(st.Core.Write(batch, nil))

	return
}

func (st *LevelDBBackend) Set(k string, v interface{}) (err error) {
	var encoded []byte
	if encoded, err = common.EncodeJSONValue(v); err != nil {
		err = setLevelDBCoreError(err)
		return
	}

	var exists bool
	if exists, err = st.Has(k); !exists || err != nil {
		if !exists {
			err = errors.ErrorStorageRecordDoesNotExist
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
				err = errors.ErrorStorageRecordDoesNotExist
				return
			}
			return
		}
	}

	batch := new(leveldb.Batch)
	for _, v := range vs {
		var encoded []byte
		if encoded, err = common.EncodeJSONValue(v); err != nil {
			err = setLevelDBCoreError(err)
			return
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
			err = errors.ErrorStorageRecordDoesNotExist
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

	if cursor != nil {
		iter.Seek(cursor)
	}

	var funcNext func() bool
	var hasUnsent bool
	if reverse {
		if !iter.Last() {
			iter.Release()
			return func() (IterItem, bool) { return IterItem{}, false }, func() {}
		}
		funcNext = iter.Prev
		hasUnsent = true
	} else {
		funcNext = iter.Next
		if cursor != nil {
			hasUnsent = true
		} else {
			hasUnsent = false
		}
	}

	var n uint64
	return func() (IterItem, bool) {
			if hasUnsent {
				hasUnsent = false
				n++
				return IterItem{N: n, Key: iter.Key(), Value: iter.Value()}, true
			}

			if !funcNext() {
				iter.Release()
				return IterItem{}, false
			}

			if limit != 0 && n >= limit {
				defer iter.Release()
				n++
				return IterItem{N: n, Key: iter.Key(), Value: iter.Value()}, false
			}
			n++
			return IterItem{N: n, Key: iter.Key(), Value: iter.Value()}, true
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

	cursor := option.Cursor
	if cursor == "" {
		cursor = prefix
	}

	var cnt uint64 = 0

	for ok := iter.Seek(st.makeKey(cursor)); ok; ok = iterFunc() {
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
