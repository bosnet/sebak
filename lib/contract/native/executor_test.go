package native

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExecutor(t *testing.T) {
	testAddress := "HELLOWORLDADDRESS"
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	var Root = sebakcommon.Hash{}
	sdb := statedb.New(Root, trie.NewEthDatabase(st))
	ctx := context.NewContext(testAddress, sdb)
	api := api.NewAPI(ctx, testAddress)

	ex := NewNativeExecutor(ctx, api)

	ex.RegisterFunc("Hello", func(ex *NativeExecutor, execCode *payload.ExecCode) (ret *value.Value, err error) {
		greeting := execCode.Args[0]
		retHello, err := ex.API().Helloworld(greeting.(string))

		ret, _ = value.ToValue(retHello)
		return
	})

	excode := &payload.ExecCode{
		ContractAddress: testAddress,
		Method:          "Hello",
		Args:            []interface{}{"boscoin"},
	}

	ret, err := ex.Execute(excode)
	if err != nil {
		t.Fatal(err)
	}
	want, err := api.Helloworld("boscoin")
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, ret.String(), want)
}
