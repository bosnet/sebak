package eth

import (
	"boscoin.io/sebak/lib/storage"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

type EthDB struct {
	ldbBackend *sebakstorage.LevelDBBackend
}

var _ ethdb.Database = (*EthDB)(nil)

func NewEthDB(ldb *sebakstorage.LevelDBBackend) *EthDB {
	return &EthDB{
		ldbBackend: ldb,
	}
}

func (db *EthDB) Put(key []byte, value []byte) error {
	return db.ldbBackend.Core.Put(key, value, nil)
}

func (db *EthDB) Has(key []byte) (bool, error) {
	return db.ldbBackend.Core.Has(key, nil)
}

func (db *EthDB) Get(key []byte) ([]byte, error) {
	dat, err := db.ldbBackend.Core.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *EthDB) Delete(key []byte) error {
	return db.ldbBackend.Core.Delete(key, nil)
}

func (db *EthDB) Close() {
	db.ldbBackend.DB.Close()
}

func (db *EthDB) NewBatch() ethdb.Batch {
	return &ldbBatch{db: db.ldbBackend, b: new(leveldb.Batch)}
}

func (db *EthDB) BackEnd() *sebakstorage.LevelDBBackend {
	return db.ldbBackend
}

type ldbBatch struct {
	db   *sebakstorage.LevelDBBackend
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
