package runner

import (
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
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

	for {
		height++

		nextBlock, err = block.GetBlockByHeight(sb.st, height)
		return
	}

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
			if err == errors.ErrorStorageRecordDoesNotExist {
				if checked {
					sb.log.Debug("stop checking; all the blocks are checked", "height", sb.checkedBlock)
				}
				err = nil
			}

			return
		}

		sb.log.Debug("check block", "block", blk)
		if err = sb.CheckByBlock(blk); err != nil {
			sb.log.Error("failed to check block", "block", blk, "height", blk.Height)
			return
		}
		sb.log.Debug("checked block", "block", blk)
		sb.checkedBlock = blk.Height
		checked = true
	}

	return
}

func (sb *SavingBlockOperations) CheckByBlock(blk block.Block) (err error) {
	for _, txHash := range blk.Transactions {
		if err = sb.CheckTransactionByBlock(blk, txHash); err != nil {
			return
		}
	}

	if blk.Height > common.GenesisBlockHeight { // ProposerTransaction
		if err = sb.CheckTransactionByBlock(blk, blk.ProposerTransaction); err != nil {
			return
		}
	}

	return
}

func (sb *SavingBlockOperations) CheckTransactionByBlock(blk block.Block, hash string) (err error) {
	var bt block.BlockTransaction
	if bt, err = block.GetBlockTransaction(sb.st, hash); err != nil {
		sb.log.Error("failed to get BlockTransaction", "block", blk, "transaction", hash)
		return
	}

	for _, op := range bt.Transaction().B.Operations {
		opHash := block.NewBlockOperationKey(op.MakeHashString(), hash)

		var exists bool
		if exists, err = block.ExistsBlockOperation(sb.st, opHash); err != nil {
			sb.log.Error(
				"failed to check ExistsBlockOperation",
				"block", blk,
				"transaction", hash,
				"operation", op,
			)
			return
		}

		if !exists {
			bt.SaveBlockOperations(sb.st, blk)
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
	sb.saveBlock <- blk
}

func (sb *SavingBlockOperations) save(blk block.Block) (err error) {
	return sb.CheckByBlock(blk)
}
