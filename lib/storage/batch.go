package storage

import (
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	leveldbIterator "github.com/syndtr/goleveldb/leveldb/iterator"
	leveldbOpt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldbUtil "github.com/syndtr/goleveldb/leveldb/util"
)

type BatchCore struct {
	sync.RWMutex

	core  LevelDBCore
	batch *leveldb.Batch

	inserted map[string][]byte
}

func NewBatchCore(core LevelDBCore) *BatchCore {
	return &BatchCore{
		core:     core,
		batch:    &leveldb.Batch{},
		inserted: map[string][]byte{},
	}
}

func (bb *BatchCore) Has(key []byte, opt *leveldbOpt.ReadOptions) (bool, error) {
	bb.RLock()
	defer bb.RUnlock()

	var found bool
	if _, found = bb.inserted[string(key)]; found {
		return true, nil
	}

	return bb.core.Has(key, opt)
}

func (bb *BatchCore) Get(key []byte, opt *leveldbOpt.ReadOptions) (b []byte, err error) {
	bb.RLock()
	defer bb.RUnlock()

	var found bool
	if b, found = bb.inserted[string(key)]; found {
		return
	}

	return bb.core.Get(key, opt)
}

// NewIterator does not work with `BatchBackend`
func (bb *BatchCore) NewIterator(r *leveldbUtil.Range, opt *leveldbOpt.ReadOptions) leveldbIterator.Iterator {
	return bb.core.NewIterator(r, opt)
}

func (bb *BatchCore) Put(key []byte, v []byte, opt *leveldbOpt.WriteOptions) error {
	bb.Lock()
	defer bb.Unlock()

	bb.inserted[string(key)] = v
	bb.batch.Put(key, v)

	return nil
}

// Write will write the existing contents of `BatchBackend.batch` and then
// argument, batch will be written.
func (bb *BatchCore) Write(batch *leveldb.Batch, opt *leveldbOpt.WriteOptions) (err error) {
	bb.Lock()
	defer bb.Unlock()

	if err = bb.core.Write(bb.batch, opt); err != nil {
		return
	}

	if batch != nil {
		err = bb.core.Write(batch, opt)
	}

	return
}

func (bb *BatchCore) Discard() {
	bb.Lock()
	defer bb.Unlock()

	bb.clear()
}

func (bb *BatchCore) Commit() (err error) {
	bb.Lock()
	defer bb.Unlock()

	err = bb.core.Write(bb.batch, nil)
	if err != nil {
		return
	}

	bb.clear()

	return
}

func (bb *BatchCore) Delete(key []byte, opt *leveldbOpt.WriteOptions) error {
	bb.Lock()
	defer bb.Unlock()

	delete(bb.inserted, string(key))
	bb.batch.Delete(key)

	return nil
}

func (bb *BatchCore) Dump() []byte {
	bb.RLock()
	defer bb.RUnlock()

	return bb.batch.Dump()
}

func (bb *BatchCore) clear() {
	bb.batch = &leveldb.Batch{}
	bb.inserted = map[string][]byte{}
}
