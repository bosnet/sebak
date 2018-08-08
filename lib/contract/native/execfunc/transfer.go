package execfunc

import (
	"boscoin.io/sebak/lib/contract/native"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
)

var TransferAddress = "transfer"

func init() {
	native.AddContract(TransferAddress, RegisterTransfer)
}

func RegisterTransfer(ex *native.NativeExecutor) {
	ex.RegisterFunc("transfer", transfer)
}

func transfer(ex *native.NativeExecutor, execCode *payload.ExecCode) (*value.Value, error) {
	//TODO(anarcher)
	/*
		sender := cfg.Transaction.Sender
		receiver := cfg.Transaction.Receiver

		transferForm(sender)
	*/

	return nil, nil
}
