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

		sb.log.Debug("check block", "block", blk)

		var st *storage.LevelDBBackend
		if st, err = sb.st.OpenBatch(); err != nil {
			break
		}

		if err = sb.CheckByBlock(st, blk); err != nil {
			sb.log.Error("failed to check block", "block", blk, "height", blk.Height)
			st.Discard()
			break
		}
		if err = st.Commit(); err != nil {
			st.Discard()
			break
		}
		sb.log.Debug("checked block", "block", blk)
		sb.checkedBlock = blk.Height
		checked = true
	}

	return
}

func (sb *SavingBlockOperations) savingBlockOperationsWorker(id int, st *storage.LevelDBBackend, blk block.Block, ops <-chan string, results chan<- error) {
	for op := range ops {
		results <- sb.CheckTransactionByBlock(st, blk, op)
	}
}

func (sb *SavingBlockOperations) CheckByBlock(st *storage.LevelDBBackend, blk block.Block) (err error) {
	ops := make(chan string, 100)
	results := make(chan error, 100)
	defer close(results)

	numWorker := int(len(blk.Transactions) / 2)
	if numWorker > 100 {
		numWorker = 100
	}

	for i := 1; i <= numWorker; i++ {
		go sb.savingBlockOperationsWorker(i, st, blk, ops, results)
	}
	for _, txHash := range blk.Transactions {
		ops <- txHash
	}
	close(ops)

	var errs []error
	for _ = range blk.Transactions {
		err = <-results
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		err = errors.FailedToSaveBlockOperaton.Clone().SetData("errors", errs)
		return
	}

	if blk.Height > common.GenesisBlockHeight { // ProposerTransaction
		if err = sb.CheckTransactionByBlock(st, blk, blk.ProposerTransaction); err != nil {
			return
		}
	}

	return
}

func (sb *SavingBlockOperations) CheckTransactionByBlock(st *storage.LevelDBBackend, blk block.Block, hash string) (err error) {
	var bt block.BlockTransaction
	if bt, err = block.GetBlockTransaction(st, hash); err != nil {
		sb.log.Error("failed to get BlockTransaction", "block", blk, "transaction", hash)
		return
	}

	if bt.Transaction().IsEmpty() {
		var tp block.TransactionPool
		if tp, err = block.GetTransactionPool(st, hash); err != nil {
			sb.log.Error("failed to get Transaction from TransactionPool", "transaction", hash)
			return
		}

		bt.Message = tp.Message
	}

	for _, op := range bt.Transaction().B.Operations {
		opHash := block.NewBlockOperationKey(op.MakeHashString(), hash)

		var exists bool
		if exists, err = block.ExistsBlockOperation(st, opHash); err != nil {
			sb.log.Error(
				"failed to check ExistsBlockOperation",
				"block", blk,
				"transaction", hash,
				"operation", op,
			)
			return
		}

		if !exists {
			bt.SaveBlockOperations(st, blk)
			sb.log.Debug("saved missing BlockOperation", "block", blk, "transaction", hash, "operation", op)
		}
	}

	return
}

func (sb *SavingBlockOperations) Start() {
	go sb.continuousCheck()
	go sb.StartSaving()

	return
}

func (sb *SavingBlockOperations) StartSaving() {
	sb.log.Debug("start saving")

	for {
		select {
		case blk := <-sb.saveBlock:
			if err := sb.save(blk); err != nil {
				// NOTE if failed, the `continuousCheck()` will fill the missings.
				sb.log.Error("failed to save BlockOperation", "block", blk, "error", err)
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
	sb.log.Debug("start to save BlockOperation", "block", blk)
	defer func() {
		sb.log.Debug("end to save BlockOperation", "block", blk, "error", err)
	}()

	var st *storage.LevelDBBackend
	if st, err = sb.st.OpenBatch(); err != nil {
		return
	}

	if err = sb.CheckByBlock(st, blk); err != nil {
		err = st.Discard()
	} else {
		err = st.Commit()
	}

	return
}
