package runner

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction/operation"
)

func getGenesisTransaction(st *storage.LevelDBBackend) (bt block.BlockTransaction, err error) {
	var bk block.Block
	if bk, err = block.GetBlockByHeight(st, common.GenesisBlockHeight); err != nil {
		return
	} else if len(bk.Transactions) < 1 {
		err = errors.ErrorWrongBlockFound
		return
	}

	if bt, err = block.GetBlockTransaction(st, bk.Transactions[0]); err != nil {
		return
	}

	if len(bt.Operations) != 2 {
		err = errors.ErrorWrongBlockFound
		return
	}

	return
}

func getGenesisAccount(st *storage.LevelDBBackend, operationIndex int) (account *block.BlockAccount, err error) {
	var bt block.BlockTransaction
	if bt, err = getGenesisTransaction(st); err != nil {
		return
	}

	var bo block.BlockOperation
	if bo, err = block.GetBlockOperation(st, bt.Operations[operationIndex]); err != nil {
		return
	}

	var opb operation.Body
	if opb, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
		return
	}
	opbp := opb.(operation.Payable)

	if account, err = block.GetBlockAccount(st, opbp.TargetAddress()); err != nil {
		return
	}

	return
}

func GetGenesisAccount(st *storage.LevelDBBackend) (account *block.BlockAccount, err error) {
	return getGenesisAccount(st, 0)
}

func GetCommonAccount(st *storage.LevelDBBackend) (account *block.BlockAccount, err error) {
	return getGenesisAccount(st, 1)
}

func GetGenesisBalance(st *storage.LevelDBBackend) (balance common.Amount, err error) {
	var bt block.BlockTransaction
	if bt, err = getGenesisTransaction(st); err != nil {
		return
	}

	var bo block.BlockOperation
	if bo, err = block.GetBlockOperation(st, bt.Operations[0]); err != nil {
		return
	}

	var opb operation.Body
	if opb, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
		return
	}
	opbp := opb.(operation.Payable)

	balance = opbp.GetAmount()

	return
}
