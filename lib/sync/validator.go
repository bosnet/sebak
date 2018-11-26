package sync

import (
	"context"
	"strconv"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"

	"github.com/inconshreveable/log15"
)

//TODO(anarcher) another name is Finisher

type BlockValidator struct {
	network   network.Network
	storage   *storage.LevelDBBackend
	txpool    *transaction.Pool
	commonCfg common.Config

	prevBlockWaitTimeout time.Duration // Waiting prev block if is doesn't exist
	logger               log15.Logger
}

type BlockValidatorOption func(*BlockValidator)

func NewBlockValidator(nw network.Network, ldb *storage.LevelDBBackend, tp *transaction.Pool, cfg common.Config, opts ...BlockValidatorOption) *BlockValidator {
	v := &BlockValidator{
		network:              nw,
		storage:              ldb,
		txpool:               tp,
		prevBlockWaitTimeout: CheckPrevBlockInterval,
		commonCfg:            cfg,

		logger: common.NopLogger(),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

func (v *BlockValidator) Validate(ctx context.Context, syncInfo *SyncInfo) error {
	exists, err := v.existsBlock(ctx, v.storage, syncInfo.Height)
	if err != nil {
		return err
	}
	if exists == true {
		v.logger.Info("this block exists", "height", syncInfo.Height)
		return nil
	}
	v.logger.Debug("start validate", "height", syncInfo.Height)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if err := v.validate(ctx, syncInfo); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return v.finishBlock(ctx, syncInfo)
	}
}

func (v *BlockValidator) validate(ctx context.Context, syncInfo *SyncInfo) error {
	//Waiting to get prev block for runner.ValidateTx
	prevBlk, err := v.getPrevBlock(ctx, syncInfo.Height)
	if err != nil {
		return err
	}

	if err := v.validateTxs(ctx, syncInfo); err != nil {
		return err
	}

	if err := v.validateBlock(ctx, syncInfo, prevBlk); err != nil {
		return err
	}

	return nil
}

func (v *BlockValidator) finishBlock(ctx context.Context, syncInfo *SyncInfo) error {
	v.logger.Debug("start finish block", "height", syncInfo.Height)

	bs, err := v.storage.OpenBatch()
	if err != nil {
		return err
	}

	if exists, err := v.existsBlock(ctx, bs, syncInfo.Height); err != nil {
		bs.Discard()
		return err
	} else if exists == true {
		v.logger.Info("This block exists", "height", syncInfo.Height)
		return nil
	}

	blk := *syncInfo.Block
	if err := blk.Save(bs); err != nil {
		if err == errors.BlockAlreadyExists {
			return nil
		}
		return err
	}

	if err := runner.FinishTransactions(blk, syncInfo.Txs, bs); err != nil {
		bs.Discard()
		return err
	}
	for _, tx := range syncInfo.Txs {
		if _, err := block.SaveTransactionPool(bs, *tx); err != nil {
			return err
		}
	}

	ptx := syncInfo.Ptx
	if err := runner.FinishProposerTransaction(bs, blk, *ptx, v.logger); err != nil {
		bs.Discard()
		return err
	}

	v.logger.Debug("finish to sync block height", "height", syncInfo.Height, "hash", blk.Hash)

	if err := bs.Commit(); err != nil {
		bs.Discard()
		return err
	}

	//clean up txs of this block in txpool.
	v.txpool.Remove(blk.Transactions...)
	v.txpool.Remove(blk.ProposerTransaction)

	select {
	case <-ctx.Done():
		return nil
	default:
		event := strconv.FormatUint(syncInfo.Height, 10)
		observer.SyncBlockWaitObserver.Trigger(event)
	}

	v.logger.Debug("end finish block", "height", syncInfo.Height)
	return nil
}

func (v *BlockValidator) validateBlock(ctx context.Context, si *SyncInfo, prevBlk *block.Block) error {
	v.logger.Debug("start validate block", "height", si.Height)
	var txs []string
	for _, tx := range si.Txs {
		txs = append(txs, tx.H.Hash)
	}

	r := voting.Basis{
		Round:     si.Block.Round,
		Height:    si.Height,
		BlockHash: prevBlk.Hash,
		TotalTxs:  si.Block.TotalTxs,
		TotalOps:  si.Block.TotalOps,
	}

	blk := block.NewBlock(si.Block.Proposer, r, si.Block.ProposerTransaction, txs, si.Block.ProposedTime)

	if blk.Hash != si.Block.Hash {
		err := errors.HashDoesNotMatch
		return err
	}

	v.logger.Debug("end validate block", "height", si.Height)

	return nil
}

func (v *BlockValidator) validateTxs(ctx context.Context, si *SyncInfo) error {
	v.logger.Debug("start validate txs", "height", si.Height)
	// proposer transaction
	if si.Ptx != nil {
		if err := si.Ptx.IsWellFormed(v.commonCfg); err != nil {
			return err
		}
	}
	// transactions
	for _, tx := range si.Txs {
		hash := tx.B.MakeHashString()
		if hash != tx.H.Hash {
			err := errors.HashDoesNotMatch
			return err
		}

		if err := tx.IsWellFormed(v.commonCfg); err != nil {
			return err
		}

		if err := runner.ValidateTx(v.storage, v.commonCfg, *tx); err != nil {
			return err
		}
	}

	v.logger.Debug("end validate txs", "height", si.Height)
	return nil
}

func (v *BlockValidator) existsBlock(ctx context.Context, st *storage.LevelDBBackend, height uint64) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		exists, err := block.ExistsBlockByHeight(st, height)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func (v *BlockValidator) getPrevBlock(pctx context.Context, height uint64) (*block.Block, error) {
	ctx, cancelFunc := context.WithCancel(pctx)

	v.logger.Debug("start waiting prev block", "height", height)

	prevHeight := height - 1
	waitC := make(chan struct{})

	go func() {
	L:
		for {
			exists, err := block.ExistsBlockByHeight(v.storage, prevHeight)
			if err != nil {
				v.logger.Error("getPrevBlock: block.ExistsBlockByHeight", "err", err)
			}
			if exists == true {
				select {
				case waitC <- struct{}{}:
				case <-ctx.Done():
				}
				break L
			}

			sleep := time.After(v.prevBlockWaitTimeout)

			select {
			case <-ctx.Done():
			case <-sleep:
			}
		}
		v.logger.Debug("end prev height db watcher", "height", height)
	}()

	event := strconv.FormatUint(prevHeight, 10)
	observer.SyncBlockWaitObserver.One(event, func(args ...interface{}) {
		select {
		case waitC <- struct{}{}:
			v.logger.Debug("try stop SyncBlockWaitObserver", "prevHeight", prevHeight)
		case <-ctx.Done():
		}
		v.logger.Debug("end SyncBlockWaitObserver", "height", height)
	})

	select {
	case <-waitC:
	case <-ctx.Done():
	}

	cancelFunc() // done observer and db watcher

	v.logger.Debug("end waiting prev block", "height", height)

	prevBlock, err := block.GetBlockByHeight(v.storage, prevHeight)
	if err != nil {
		v.logger.Error("prevBlock: GetBlockByHeight", "prevHeight", prevHeight, "err", err)
		return nil, err
	}

	return &prevBlock, nil
}
