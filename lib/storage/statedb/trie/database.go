package trie

import (
	"boscoin.io/sebak/lib/storage"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
)

type EthDatabase struct {
	ldbBackend *storage.LevelDBBackend
	quitLock   sync.Mutex // Mutex protecting the quit channel access
}

func NewEthDatabase(ldb *storage.LevelDBBackend) *EthDatabase {
	return &EthDatabase{
		ldbBackend: ldb,
	}
}

func (db *EthDatabase) Put(key []byte, value []byte) error {
	return db.ldbBackend.Core.Put(key, value, nil)
}

func (db *EthDatabase) Has(key []byte) (bool, error) {
	return db.ldbBackend.Core.Has(key, nil)
}

func (db *EthDatabase) Get(key []byte) ([]byte, error) {
	dat, err := db.ldbBackend.Core.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *EthDatabase) Delete(key []byte) error {
	return db.ldbBackend.Core.Delete(key, nil)
}

func (db *EthDatabase) Close() {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()
	db.ldbBackend.DB.Close()
}

func (db *EthDatabase) NewBatch() ethdb.Batch {
	return &ldbBatch{db: db.ldbBackend, b: new(leveldb.Batch)}
}

func (db *EthDatabase) BackEnd() *storage.LevelDBBackend {
	return db.ldbBackend
}

type ldbBatch struct {
	db   *storage.LevelDBBackend
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size += 1
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Core.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}
