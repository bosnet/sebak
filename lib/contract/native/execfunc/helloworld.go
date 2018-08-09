package execfunc

import (
	"encoding/json"

	"boscoin.io/sebak/lib/contract/native"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/store/statestore"
)

var HelloWorldAddress = "HELLOWORLDADDRESS"

func init() {
	native.AddContract(HelloWorldAddress, RegisterHelloWorld)
}

func RegisterHelloWorld(ex *native.NativeExecutor) {
	ex.RegisterFunc("hello", hello)
}

func hello(ex *native.NativeExecutor, execCode *payload.ExecCode) (*value.Value, error) {
	stateClone := ex.Context.StateClone
	sender := ex.Context.SenderAccount

	item, err := stateClone.GetStorageItem(HelloWorldAddress, "greeters")
	if err != nil {
		return nil, err
	}

	var greeters []string // sebak.BlockAccount.Address is string so..
	if item == nil {
		item = &statestore.StorageItem{}
	} else {
		err := json.Unmarshal(item.Value, greeters)
		if err != nil {
			return nil, err
		}
	}

	greeters = append(greeters, sender.Address)

	{
		b, err := json.Marshal(greeters)
		if err != nil {
			return nil, err
		}
		item.Value = b
	}

	if err := stateClone.PutStorageItem(HelloWorldAddress, "greeters", item); err != nil {
		return nil, err
	}

	v, _ := value.ToValue("world")
	rCode := v
	return rCode, nil
}
