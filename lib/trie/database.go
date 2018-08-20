package trie

import (
	"boscoin.io/sebak/lib/storage"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
)

type EthDatabase struct {
	ldbBackend *sebakstorage.LevelDBBackend
	quitLock   sync.Mutex // Mutex protecting the quit channel access
}

func NewEthDatabase(ldb *sebakstorage.LevelDBBackend) *EthDatabase {
	return &EthDatabase{
		ldbBackend: ldb,
	}
}

func (db *EthDatabase) Put(key []byte, value []byte) error {
	return db.ldbBackend.DB.Put(key, value, nil)
}

func (db *EthDatabase) Has(key []byte) (bool, error) {
	return db.ldbBackend.DB.Has(key, nil)
}

func (db *EthDatabase) Get(key []byte) ([]byte, error) {
	dat, err := db.ldbBackend.DB.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *EthDatabase) Delete(key []byte) error {
	return db.ldbBackend.DB.Delete(key, nil)
}

func (db *EthDatabase) Close() {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()
	db.ldbBackend.DB.Close()
}

func (db *EthDatabase) NewBatch() ethdb.Batch {
	return &ldbBatch{db: db.ldbBackend.DB, b: new(leveldb.Batch)}
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}
