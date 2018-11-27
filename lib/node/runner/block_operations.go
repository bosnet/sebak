package runner

import (
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
)

type SavingBlockOperations struct {
	st  *storage.LevelDBBackend
	log logging.Logger

	saveBlock    chan block.Block
	checkedBlock uint64 // block.Block.Height
}

func NewSavingBlockOperations(st *storage.LevelDBBackend, logger logging.Logger) *SavingBlockOperations {
	if logger == nil {
		logger = log
	}

	logger = logger.New(logging.Ctx{"m": "SavingBlockOperations"})
	return &SavingBlockOperations{
		st:           st,
		log:          logger,
		saveBlock:    make(chan block.Block, 10),
		checkedBlock: common.GenesisBlockHeight,
	}
}

func (sb *SavingBlockOperations) getNextBlock(height uint64) (nextBlock block.Block, err error) {
	if height < sb.checkedBlock {
		height = sb.checkedBlock
	}

	height++

	nextBlock, err = block.GetBlockByHeight(sb.st, height)

	return
}

func (sb *SavingBlockOperations) Check() (err error) {
	sb.log.Debug("start to check")
	return sb.check()
}

// continuousCheck will check the missing `BlockOperation`s continuously; if it
// is failed, still try.
func (sb *SavingBlockOperations) continuousCheck() {
	for {
		if err := sb.check(); err != nil {
			sb.log.Error("failed to check", "error", err)
		}

		time.Sleep(5 * time.Second)
	}
}

// check checks whether `BlockOperation`s of latest `block.Block` are saved; if
// not it will try to catch up to the last.
func (sb *SavingBlockOperations) check() (err error) {
	var checked bool
	var blk block.Block
	for {
		if blk, err = sb.getNextBlock(blk.Height); err != nil {
			if err == errors.StorageRecordDoesNotExist {
				if checked {
					sb.log.Debug("stop checking; all the blocks are checked", "height", sb.checkedBlock)
				}
				err = nil
			}

			break
		}

		sb.log.Debug("check block", "block", blk.Hash)

		var st *storage.LevelDBBackend
		if st, err = sb.st.OpenBatch(); err != nil {
			break
		}

		if err = sb.CheckByBlock(st, blk); err != nil {
			sb.log.Error("failed to check block", "block", blk.Hash, "height", blk.Height, "error", err)
			st.Discard()
			break
		}
		if err = st.Commit(); err != nil {
			sb.log.Error("failed to commit", "block", blk.Hash, "height", blk.Height, "error", err)
			st.Discard()
			break
		}
		sb.log.Debug("checked block", "block", blk.Hash)
		sb.checkedBlock = blk.Height
		checked = true
	}

	return
}

func (sb *SavingBlockOperations) savingBlockOperationsWorker(id int, st *storage.LevelDBBackend, blk block.Block, txs <-chan string, errChan chan<- error) {
	for hash := range txs {
		errChan <- sb.CheckTransactionByBlock(st, blk, hash)
	}
}

func (sb *SavingBlockOperations) CheckByBlock(st *storage.LevelDBBackend, blk block.Block) (err error) {
	if blk.Height > common.GenesisBlockHeight { // ProposerTransaction
		if err = sb.CheckTransactionByBlock(st, blk, blk.ProposerTransaction); err != nil {
			return
		}
	}
	if len(blk.Transactions) < 1 {
		return
	}

	txs := make(chan string, 100)
	errChan := make(chan error, 100)
	defer close(errChan)

	numWorker := int(len(blk.Transactions) / 2)
	if numWorker > 100 {
		numWorker = 100
	} else if numWorker < 1 {
		numWorker = 1
	}

	for i := 1; i <= numWorker; i++ {
		go sb.savingBlockOperationsWorker(i, st, blk, txs, errChan)
	}

	go func() {
		for _, hash := range blk.Transactions {
			txs <- hash
		}
		close(txs)
	}()

	var errs []error
	var returned int
errorCheck:
	for {
		select {
		case err = <-errChan:
			returned++
			if err != nil {
				errs = append(errs, err)
			}
			if returned == len(blk.Transactions) {
				break errorCheck
			}
		}
	}

	if len(errs) > 0 {
		err = errors.FailedToSaveBlockOperaton.Clone().SetData("errors", errs)
		return
	}

	return
}

func (sb *SavingBlockOperations) CheckTransactionByBlock(st *storage.LevelDBBackend, blk block.Block, hash string) (err error) {
	var bt block.BlockTransaction
	if bt, err = block.GetBlockTransaction(st, hash); err != nil {
		sb.log.Error("failed to get BlockTransaction", "block", blk.Hash, "transaction", hash, "error", err)
		return
	}

	if bt.Transaction().IsEmpty() {
		var tp block.TransactionPool
		if tp, err = block.GetTransactionPool(st, hash); err != nil {
			sb.log.Error("failed to get Transaction from TransactionPool", "transaction", hash, "error", err)
			return
		}

		bt.Message = tp.Message
	}

	for i, op := range bt.Transaction().B.Operations {
		opHash := block.NewBlockOperationKey(op.MakeHashString(), hash)

		var exists bool
		if exists, err = block.ExistsBlockOperation(st, opHash); err != nil {
			sb.log.Error(
				"failed to check ExistsBlockOperation",
				"block", blk.Hash,
				"transaction", hash,
				"operation-index", i,
			)
			return
		}

		if !exists {
			if err = bt.SaveBlockOperation(st, op); err != nil {
				return err
			}
		}
	}

	return
}

func (sb *SavingBlockOperations) Start() {
	go sb.continuousCheck()
	go sb.startSaving()

	return
}

func (sb *SavingBlockOperations) startSaving() {
	sb.log.Debug("start saving")

	for {
		select {
		case blk := <-sb.saveBlock:
			if err := sb.save(blk); err != nil {
				// NOTE if failed, the `continuousCheck()` will fill the missings.
				sb.log.Error("failed to save BlockOperation", "block", blk.Hash, "error", err)
			}
		}
	}
}

func (sb *SavingBlockOperations) Save(blk block.Block) {
	go func() {
		sb.saveBlock <- blk
	}()
}

func (sb *SavingBlockOperations) save(blk block.Block) (err error) {
	sb.log.Debug("starting to save BlockOperation", "block", blk.Hash)
	defer func() {
		if err != nil {
			sb.log.Error("could not save BlockOperation", "block", blk, "error", err)
		} else {
			sb.log.Debug("done saving BlockOperation", "block", blk.Hash)
		}
	}()

	var st *storage.LevelDBBackend
	if st, err = sb.st.OpenBatch(); err != nil {
		return
	}

	if err = sb.CheckByBlock(st, blk); err != nil {
		st.Discard()
	} else {
		err = st.Commit()
	}

	return
}
