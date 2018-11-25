//
// Provides a replacement for LevelDBBackend suitable for unit tests
//
// LevelDB allows one to create a memory DB where we can store test
// data during our unit tests
//
package storage

//
// Returns:
//  A new memory DB
//
func NewTestStorage() *LevelDBBackend {
	st := &LevelDBBackend{}
	config, _ := NewConfigFromString("memory://")
	if err := st.Init(config); err != nil {
		panic(err)
	}

	return st
}
