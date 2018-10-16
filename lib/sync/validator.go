package sync

import (
	"context"
	"fmt"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
)

//TODO(anarcher) another name is Finisher

type BlockValidator struct {
	network   network.Network
	storage   *storage.LevelDBBackend
	commonCfg common.Config

	networkID []byte

	prevBlockWaitTimeout time.Duration // Waiting prev block if is doesn't exist
	logger               log15.Logger
}

type BlockValidatorOption func(*BlockValidator)

func NewBlockValidator(nw network.Network, ldb *storage.LevelDBBackend, networkID []byte, cfg common.Config, opts ...BlockValidatorOption) *BlockValidator {
	v := &BlockValidator{
		network:              nw,
		storage:              ldb,
		networkID:            networkID,
		prevBlockWaitTimeout: 200 * time.Millisecond,
		commonCfg:            cfg,

		logger: NopLogger(),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

func (v *BlockValidator) Validate(ctx context.Context, syncInfo *SyncInfo) error {
	exists, err := v.existsBlock(ctx, v.storage, syncInfo.BlockHeight)
	if err != nil {
		return err
	}
	if exists == true {
		v.logger.Info("This block exists", "height", syncInfo.BlockHeight)
		return nil
	}

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
	prevBlk, err := v.getPrevBlock(ctx, syncInfo.BlockHeight)
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
	ts, err := v.storage.OpenTransaction()
	if err != nil {
		return err
	}

	if exists, err := v.existsBlock(ctx, ts, syncInfo.BlockHeight); err != nil {
		ts.Discard()
		return err
	} else if exists == true {
		v.logger.Info("This block exists", "height", syncInfo.BlockHeight)
		return nil
	}

	//TODO(anarcher): using leveldb.Tx or leveldb.Batch?
	blk := *syncInfo.Block
	if err := blk.Save(ts); err != nil {
		if err == errors.ErrorBlockAlreadyExists {
			return nil
		}
		return err
	}

	if err = runner.FinishTransactions(blk, syncInfo.Txs, ts); err != nil {
		ts.Discard()
		return err
	}
	v.logger.Info(fmt.Sprintf("finish to sync block height: %v", syncInfo.BlockHeight), "height", syncInfo.BlockHeight, "hash", blk.Hash)

	if err := ts.Commit(); err != nil {
		ts.Discard()
		return err
	}

	select {
	case <-ctx.Done():
		return nil
	default:
		observer.SyncBlockWaitObserver.Trigger(string(syncInfo.BlockHeight))
	}

	return nil
}

func (v *BlockValidator) validateBlock(ctx context.Context, si *SyncInfo, prevBlk *block.Block) error {
	var txs []string
	for _, tx := range si.Txs {
		txs = append(txs, tx.H.Hash)
	}

	round := si.Block.Round
	round.BlockHash = prevBlk.Hash

	blk := block.NewBlock(si.Block.Proposer, si.Block.Round, si.Block.ProposerTransaction, txs, si.Block.Confirmed)

	if blk.Hash != si.Block.Hash {
		err := errors.ErrorHashDoesNotMatch
		return err
	}

	return nil
}

func (v *BlockValidator) validateTxs(ctx context.Context, si *SyncInfo) error {
	for _, tx := range si.Txs {
		hash := tx.B.MakeHashString()
		if hash != tx.H.Hash {
			err := errors.ErrorHashDoesNotMatch
			return err
		}

		if err := tx.IsWellFormed(v.networkID, v.commonCfg); err != nil {
			return err
		}

		if err := runner.ValidateTx(v.storage, *tx); err != nil {
			return err
		}
	}
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
	ctx, cancelFunc := context.WithTimeout(pctx, v.prevBlockWaitTimeout)
	defer cancelFunc()

	prevHeight := height - 1

	exists, err := block.ExistsBlockByHeight(v.storage, prevHeight)
	if err != nil {
		return nil, err
	}

	if exists == false {
		waitC := make(chan struct{})
		observer.SyncBlockWaitObserver.On(string(prevHeight), func(args ...interface{}) {
			select {
			case waitC <- struct{}{}:
			case <-ctx.Done():
			}
		})

		select {
		case <-waitC:
		case <-ctx.Done():
		}
	}

	prevBlock, err := block.GetBlockByHeight(v.storage, prevHeight)
	if err != nil {
		return nil, err
	}

	return &prevBlock, nil
}
