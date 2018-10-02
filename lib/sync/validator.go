package sync

import (
	"context"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
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

	logger log15.Logger
}

type BlockValidatorOption func(*BlockValidator)

func NewBlockValidator(nw network.Network, ldb *storage.LevelDBBackend, networkID []byte, cfg common.Config, opts ...BlockValidatorOption) *BlockValidator {
	v := &BlockValidator{
		network:   nw,
		storage:   ldb,
		networkID: networkID,
		commonCfg: cfg,

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
	if err := v.validateTxs(ctx, syncInfo); err != nil {
		return err
	}

	if err := v.validateBlock(ctx, syncInfo); err != nil {
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
	v.logger.Info("Finish to sync block", "height", syncInfo.BlockHeight)

	if err := ts.Commit(); err != nil {
		ts.Discard()
		return err
	}

	return nil
}

func (v *BlockValidator) validateBlock(ctx context.Context, si *SyncInfo) error {
	var txs []string
	for _, tx := range si.Txs {
		txs = append(txs, tx.H.Hash)
	}
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
