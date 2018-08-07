package jsvm

import (
	"fmt"
	"os"
	"testing"

	"boscoin.io/sebak/lib/store/statestore"

	"boscoin.io/sebak/lib/storage"
)

var (
	testStateStore     *statestore.StateStore
	testStateClone     *statestore.StateClone
	testLevelDBBackend *sebakstorage.LevelDBBackend
	testAddress = "testaddress"
	testCode = `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`
)

func TestMain(m *testing.M) {
	var err error

	config, err := sebakstorage.NewConfigFromString("memory://")
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}
	testLevelDBBackend, err = sebakstorage.NewStorage(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}

	testStateStore = statestore.NewStateStore(testLevelDBBackend)
	testStateClone = statestore.NewStateClone(testStateStore)

	m.Run()

	err = testLevelDBBackend.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}

}
