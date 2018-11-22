package api

import (
	"bytes"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/http/httptest"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-unittest")

const (
	QueryPattern = "cursor={cursor}&limit={limit}&reverse={reverse}&type={type}"
)

func prepareAPIServer() (*httptest.Server, *storage.LevelDBBackend) {
	storage := block.InitTestBlockchain()
	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountTransactionsHandlerPattern, apiHandler.GetTransactionsByAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountOperationsHandlerPattern, apiHandler.GetOperationsByAccountHandler).Methods("GET")
	router.HandleFunc(GetTransactionsHandlerPattern, apiHandler.GetTransactionsHandler).Methods("GET")
	router.HandleFunc(GetTransactionByHashHandlerPattern, apiHandler.GetTransactionByHashHandler).Methods("GET")
	router.HandleFunc(GetTransactionStatusHandlerPattern, apiHandler.GetTransactionStatusByHashHandler).Methods("GET")
	router.HandleFunc(GetTransactionOperationsHandlerPattern, apiHandler.GetOperationsByTxHashHandler).Methods("GET")
	router.HandleFunc(GetBlocksHandlerPattern, apiHandler.GetBlocksHandler).Methods("GET")
	router.HandleFunc(GetBlockHandlerPattern, apiHandler.GetBlockHandler).Methods("GET")
	router.HandleFunc(PostSubscribePattern, apiHandler.PostSubscribeHandler).Methods("POST")
	ts := httptest.NewServer(router)
	return ts, storage
}

func prepareOps(storage *storage.LevelDBBackend, count int) (*keypair.Full, []block.BlockOperation) {
	kp, btList := prepareTxs(storage, count)
	var boList []block.BlockOperation
	for _, bt := range btList {
		bo, err := block.GetBlockOperation(storage, bt.Operations[0])
		if err != nil {
			panic(err)
		}
		boList = append(boList, bo)
	}

	return kp, boList
}
func prepareOpsWithoutSave(count int, st *storage.LevelDBBackend) (*keypair.Full, block.Block, []block.BlockOperation) {
	kp := keypair.Random()
	var txs []transaction.Transaction
	var txHashes []string
	var boList []block.BlockOperation
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlockWithPrevBlock(block.GetLatestBlock(st), txHashes)
	for _, tx := range txs {
		for _, op := range tx.B.Operations {
			bo, err := block.NewBlockOperationFromOperation(op, tx, theBlock.Height)
			if err != nil {
				panic(err)
			}
			boList = append(boList, bo)
		}
	}

	return kp, theBlock, boList
}

func prepareBlkTxOpWithoutSave(st *storage.LevelDBBackend) (*keypair.Full, block.Block, block.BlockTransaction, block.BlockOperation) {
	kp := keypair.Random()
	var txHashes []string
	tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
	txHashes = append(txHashes, tx.GetHash())
	theBlock := block.TestMakeNewBlockWithPrevBlock(block.GetLatestBlock(st), txHashes)
	bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, theBlock.ProposedTime, tx)

	op := tx.B.Operations[0]
	bo, err := block.NewBlockOperationFromOperation(op, tx, theBlock.Height)
	if err != nil {
		panic(err)
	}

	return kp, theBlock, bt, bo
}

func prepareTxs(storage *storage.LevelDBBackend, count int) (*keypair.Full, []block.BlockTransaction) {
	kp := keypair.Random()
	var txs []transaction.Transaction
	var txHashes []string
	var btList []block.BlockTransaction
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlockWithPrevBlock(block.GetLatestBlock(storage), txHashes)
	theBlock.MustSave(storage)
	for _, tx := range txs {
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, theBlock.ProposedTime, tx)
		bt.MustSave(storage)
		if err := bt.SaveBlockOperations(storage); err != nil {
			return nil, nil
		}
		btList = append(btList, bt)
	}
	return kp, btList
}

func prepareTxsWithoutSave(count int, st *storage.LevelDBBackend) (*keypair.Full, []block.BlockTransaction) {
	kp := keypair.Random()
	var txs []transaction.Transaction
	var txHashes []string
	var btList []block.BlockTransaction
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlockWithPrevBlock(block.GetLatestBlock(st), txHashes)
	for _, tx := range txs {
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, theBlock.ProposedTime, tx)
		btList = append(btList, bt)
	}
	return kp, btList
}

func prepareTxWithoutSave(st *storage.LevelDBBackend) (*keypair.Full, *transaction.Transaction, *block.BlockTransaction) {
	kp := keypair.Random()
	tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)

	theBlock := block.TestMakeNewBlockWithPrevBlock(block.GetLatestBlock(st), []string{tx.GetHash()})
	bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, theBlock.ProposedTime, tx)
	return kp, &tx, &bt
}

func request(ts *httptest.Server, url string, streaming bool, bodys ...[]byte) io.ReadCloser {
	// Do a Request
	url = ts.URL + url
	var req *http.Request
	if len(bodys) == 0 {
		var err error
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			panic(err)
		}
	} else {
		var err error
		req, err = http.NewRequest("POST", url, bytes.NewReader(bodys[0]))
		if err != nil {
			panic(err)
		}
	}
	if streaming {
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		panic(err)
	}
	return resp.Body
}
