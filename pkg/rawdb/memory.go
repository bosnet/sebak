package rawdb

type MemoryDb struct {
	db map[string][]byte
}

func NewMemoryDb() *MemoryDb {
	return &MemoryDb{
		db: make(map[string][]byte),
	}
}

func (o *MemoryDb) Put(key []byte, value []byte) error {
	o.db[string(key)] = value
	return nil
}

func (o *MemoryDb) Has(key []byte) (bool, error) {
	_, ok := o.db[string(key)]
	return ok, nil
}

func (o *MemoryDb) Get(key []byte) ([]byte, error) {
	if v, ok := o.db[string(key)]; ok {
		return v, nil
	}
	return nil, nil
}

func (o *MemoryDb) Delete(key []byte) error {
	delete(o.db, string(key))
	return nil
}
