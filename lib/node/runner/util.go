package runner

import (
	"sync"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/version"
)

func GetGenesisTransaction(st *storage.LevelDBBackend) (bt block.BlockTransaction, err error) {
	var bk block.Block
	if bk, err = block.GetBlockByHeight(st, common.GenesisBlockHeight); err != nil {
		return
	}

	if len(bk.Transactions) < 1 {
		err = errors.WrongBlockFound
		return
	}

	if bt, err = block.GetBlockTransaction(st, bk.Transactions[0]); err != nil {
		return
	}

	if len(bt.Operations) != 2 {
		err = errors.WrongBlockFound
		return
	}

	return
}

func getGenesisAccount(st *storage.LevelDBBackend, operationIndex int) (account *block.BlockAccount, err error) {
	var bt block.BlockTransaction
	if bt, err = GetGenesisTransaction(st); err != nil {
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
	if bt, err = GetGenesisTransaction(st); err != nil {
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

func NewNodeInfo(nr *NodeRunner) node.NodeInfo {
	localNode := nr.Node()

	var endpoint *common.Endpoint
	if localNode.PublishEndpoint() != nil {
		endpoint = localNode.PublishEndpoint()
	}

	nv := node.NodeVersion{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		GitState:  version.GitState,
		BuildDate: version.BuildDate,
	}

	nd := node.NodeInfoNode{
		Version:    nv,
		State:      localNode.State(),
		Alias:      localNode.Alias(),
		Address:    localNode.Address(),
		Endpoint:   endpoint,
		Validators: localNode.GetValidators(),
	}

	policy := node.NodePolicy{
		NetworkID:                 string(nr.NetworkID()),
		InitialBalance:            nr.InitialBalance,
		BaseReserve:               common.BaseReserve,
		BaseFee:                   common.BaseFee,
		BlockTime:                 nr.Conf.BlockTime,
		BlockTimeDelta:            nr.Conf.BlockTimeDelta,
		TimeoutINIT:               nr.Conf.TimeoutINIT,
		TimeoutSIGN:               nr.Conf.TimeoutSIGN,
		TimeoutACCEPT:             nr.Conf.TimeoutACCEPT,
		TimeoutALLCONFIRM:         nr.Conf.TimeoutALLCONFIRM,
		RateLimitRuleAPI:          nr.Conf.RateLimitRuleAPI.Default.Formatted,
		RateLimitRuleNode:         nr.Conf.RateLimitRuleNode.Default.Formatted,
		OperationsLimit:           nr.Conf.OpsLimit,
		TransactionsLimit:         nr.Conf.TxsLimit,
		OperationsInBallotLimit:   nr.Conf.OpsInBallotLimit,
		GenesisBlockConfirmedTime: common.GenesisBlockConfirmedTime,
		InflationRatio:            common.InflationRatioString,
		UnfreezingPeriod:          common.UnfreezingPeriod,
		BlockHeightEndOfInflation: common.BlockHeightEndOfInflation,
	}

	return node.NodeInfo{
		Node:   nd,
		Policy: policy,
	}
}

type TransactionCache struct {
	sync.RWMutex

	st    *storage.LevelDBBackend
	pool  *transaction.Pool
	cache map[string]transaction.Transaction
}

func NewTransactionCache(st *storage.LevelDBBackend, pool *transaction.Pool) *TransactionCache {
	return &TransactionCache{
		st:    st,
		pool:  pool,
		cache: map[string]transaction.Transaction{},
	}
}

func (b *TransactionCache) Get(hash string) (tx transaction.Transaction, found bool, err error) {
	b.RLock()
	tx, found = b.cache[hash]
	b.RUnlock()

	if found {
		return
	}

	b.Lock()
	defer b.Unlock()

	tx, found = b.pool.Get(hash)
	if found {
		b.cache[hash] = tx
		return
	}

	if found, err = block.ExistsTransactionPool(b.st, hash); err != nil {
		return
	} else if !found {
		return
	}

	var tp block.TransactionPool
	if tp, err = block.GetTransactionPool(b.st, hash); err != nil {
		return
	}
	tx = tp.Transaction()
	b.cache[hash] = tx

	return
}
