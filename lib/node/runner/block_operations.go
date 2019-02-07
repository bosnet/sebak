package runner

import (
	"fmt"
	"sync"
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

	saveBlock          chan block.Block
	checkedBlockHeight uint64 // block.Block.Height
}

func NewSavingBlockOperations(st *storage.LevelDBBackend, logger logging.Logger) *SavingBlockOperations {
	if logger == nil {
		logger = log
	}

	logger = logger.New(logging.Ctx{"m": "SavingBlockOperations"})
	sb := &SavingBlockOperations{
		st:        st,
		log:       logger,
		saveBlock: make(chan block.Block, 10),
	}
	sb.checkedBlockHeight = sb.getCheckedBlockHeight()
	sb.log.Debug("last checked block is", "height", sb.checkedBlockHeight)

	return sb
}

func (sb *SavingBlockOperations) getCheckedBlockKey() string {
	return fmt.Sprintf("%s-last-checked-block", common.InternalPrefix)
}

func (sb *SavingBlockOperations) getCheckedBlockHeight() uint64 {
	var checked uint64
	if err := sb.st.Get(sb.getCheckedBlockKey(), &checked); err != nil {
		sb.log.Error("failed to check CheckedBlock", "error", err)
		return common.GenesisBlockHeight
	}

	return checked
}

func (sb *SavingBlockOperations) saveCheckedBlock(height uint64) {
	sb.log.Debug("save CheckedBlock", "height", height)

	var found bool
	var err error
	if found, err = sb.st.Has(sb.getCheckedBlockKey()); err != nil {
		sb.log.Error("failed to get CheckedBlock", "error", err)
		return
	}

	if !found {
		if err = sb.st.New(sb.getCheckedBlockKey(), height); err != nil {
			sb.log.Error("failed to save new CheckedBlock", "error", err)
			return
		}
	} else {
		if err = sb.st.Set(sb.getCheckedBlockKey(), height); err != nil {
			sb.log.Error("failed to set CheckedBlock", "error", err)
			return
		}
	}
	sb.checkedBlockHeight = height

	return
}

func (sb *SavingBlockOperations) getNextBlock(height uint64) (nextBlock block.Block, err error) {
	if height < sb.checkedBlockHeight {
		height = sb.checkedBlockHeight
	}

	height++

	nextBlock, err = block.GetBlockByHeight(sb.st, height)

	return
}

func (sb *SavingBlockOperations) Check() (err error) {
	sb.log.Debug("start to SavingBlockOperations.Check()", "height", sb.checkedBlockHeight)

	defer func() {
		sb.log.Debug("finished to check")
	}()

	return sb.check(sb.checkedBlockHeight)
}

// continuousCheck will check the missing `BlockOperation`s continuously; if it
// is failed, still try.
func (sb *SavingBlockOperations) continuousCheck() {
	for {
		if err := sb.check(sb.checkedBlockHeight); err != nil {
			sb.log.Error("failed to check", "error", err)
		}

		time.Sleep(5 * time.Second)
	}
}

func (sb *SavingBlockOperations) checkBlockWorker(id int, blocks <-chan block.Block, errChan chan<- error) {
	var err error
	var st *storage.LevelDBBackend

	for blk := range blocks {
		if st, err = sb.st.OpenBatch(); err != nil {
			errChan <- err
			return
		}

		if err = sb.CheckByBlock(st, blk); err != nil {
			st.Discard()
			errChan <- err
			return
		}

		if err = st.Commit(); err != nil {
			sb.log.Error("failed to commit", "block", blk.Hash, "height", blk.Height, "error", err)
			st.Discard()
			errChan <- err
			return
		}

		errChan <- nil
	}
}

// check checks whether `BlockOperation`s of latest `block.Block` are saved; if
// not it will try to catch up to the last.
func (sb *SavingBlockOperations) check(startBlockHeight uint64) (err error) {
	latestBlockHeight := block.GetLatestBlock(sb.st).Height
	if latestBlockHeight == common.GenesisBlockHeight {
		return
	}
	if latestBlockHeight <= startBlockHeight {
		return
	}

	blocks := make(chan block.Block, 100)
	errChan := make(chan error, 100)
	defer close(errChan)
	defer close(blocks)

	numWorker := int((latestBlockHeight - startBlockHeight) / 2)
	if numWorker > 100 {
		numWorker = 100
	} else if numWorker < 1 {
		numWorker = 1
	}

	for i := 1; i <= numWorker; i++ {
		go sb.checkBlockWorker(i, blocks, errChan)
	}

	var lock sync.Mutex
	closed := false
	defer func() {
		lock.Lock()
		defer lock.Unlock()

		closed = true
	}()

	go func() {
		var height uint64 = startBlockHeight
		var blk block.Block
		for {
			if blk, err = sb.getNextBlock(height); err != nil {
				err = errors.FailedToSaveBlockOperaton.Clone().SetData("error", err)
				errChan <- err
				return
			}

			if closed {
				break
			}

			blocks <- blk
			height = blk.Height
			if blk.Height == latestBlockHeight {
				break
			}
		}
	}()

	var errs uint64
errorCheck:
	for {
		select {
		case e := <-errChan:
			errs++
			if e != nil {
				err = e
				break errorCheck
			}
			if errs == (latestBlockHeight - startBlockHeight) {
				break errorCheck
			}
		}
	}

	if err != nil {
		err = errors.FailedToSaveBlockOperaton.Clone().SetData("error", err)
	} else {
		sb.saveCheckedBlock(latestBlockHeight)
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
		opHash := block.NewBlockOperationKey(common.MustMakeObjectHashString(op), hash)

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
