package api

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"io"
	"net/http"
	"net/http/httptest"
)

var networkID []byte = []byte("sebak-test-network")

const (
	QueryPattern = "cursor={cursor}&limit={limit}&reverse={reverse}&type={type}"
)

func prepareAPIServer() (*httptest.Server, *storage.LevelDBBackend, error) {
	storage, err := storage.NewTestMemoryLevelDBBackend()
	if err != nil {
		return nil, nil, err
	}

	apiHandler := NetworkHandlerAPI{storage: storage}

	router := mux.NewRouter()
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountTransactionsHandlerPattern, apiHandler.GetTransactionsByAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountOperationsHandlerPattern, apiHandler.GetOperationsByAccountHandler).Methods("GET")
	router.HandleFunc(GetTransactionsHandlerPattern, apiHandler.GetTransactionsHandler).Methods("GET")
	router.HandleFunc(GetTransactionByHashHandlerPattern, apiHandler.GetTransactionByHashHandler).Methods("GET")
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")
	router.HandleFunc(GetAccountHandlerPattern, apiHandler.GetAccountHandler).Methods("GET")
	ts := httptest.NewServer(router)
	return ts, storage, nil
}

func prepareOps(storage *storage.LevelDBBackend, blockHeight uint64, count int, kp *keypair.Full) (*keypair.Full, []block.BlockOperation, error) {
	kp, btList, err := prepareTxs(storage, blockHeight, count, kp)
	if err != nil {
		return nil, nil, err
	}
	var boList []block.BlockOperation
	for _, bt := range btList {
		bo, err := block.GetBlockOperation(storage, bt.Operations[0])
		if err != nil {
			return nil, nil, err
		}
		boList = append(boList, bo)
	}

	return kp, boList, nil
}
func prepareOpsWithoutSave(blockHeight uint64, count int, kp *keypair.Full) (*keypair.Full, []block.BlockOperation, error) {

	var err error
	if kp == nil {
		kp, err = keypair.Random()
		if err != nil {
			return nil, nil, err
		}
	}
	var txs []transaction.Transaction
	var txHashes []string
	var boList []block.BlockOperation
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	theBlock.Height += blockHeight
	for _, tx := range txs {
		for _, op := range tx.B.Operations {
			bo := block.NewBlockOperationFromOperation(op, tx, theBlock.Height)
			boList = append(boList, bo)
		}
	}

	return kp, boList, nil
}

func prepareTxs(storage *storage.LevelDBBackend, blockHeight uint64, count int, kp *keypair.Full) (*keypair.Full, []block.BlockTransaction, error) {
	var err error
	if kp == nil {
		kp, err = keypair.Random()
		if err != nil {
			return nil, nil, err
		}
	}
	var txs []transaction.Transaction
	var txHashes []string
	var btList []block.BlockTransaction
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	theBlock.Height += blockHeight
	err = theBlock.Save(storage)
	if err != nil {
		return nil, nil, err
	}
	for _, tx := range txs {
		a, err := tx.Serialize()
		if err != nil {
			return nil, nil, err
		}
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
		err = bt.Save(storage)
		if err != nil {
			return nil, nil, err
		}
		btList = append(btList, bt)
	}
	return kp, btList, nil
}

func prepareTxsWithoutSave(blockHeight uint64, count int, kp *keypair.Full) (*keypair.Full, []block.BlockTransaction, error) {
	var err error
	if kp == nil {
		kp, err = keypair.Random()
		if err != nil {
			return nil, nil, err
		}
	}
	var txs []transaction.Transaction
	var txHashes []string
	var btList []block.BlockTransaction
	for i := 0; i < count; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	theBlock := block.TestMakeNewBlock(txHashes)
	theBlock.Height += blockHeight
	for _, tx := range txs {
		a, err := tx.Serialize()
		if err != nil {
			return nil, nil, err
		}
		bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
		btList = append(btList, bt)
	}
	return kp, btList, nil
}

func prepareTxWithoutSave() (*keypair.Full, *transaction.Transaction, *block.BlockTransaction, error) {
	kp, err := keypair.Random()
	if err != nil {
		return nil, nil, nil, err
	}

	tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
	a, err := tx.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}

	theBlock := block.TestMakeNewBlock([]string{tx.GetHash()})
	bt := block.NewBlockTransactionFromTransaction(theBlock.Hash, theBlock.Height, tx, a)
	return kp, &tx, &bt, nil
}

func request(ts *httptest.Server, url string, streaming bool) (io.ReadCloser, error) {
	// Do a Request
	url = ts.URL + url
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if streaming {
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
