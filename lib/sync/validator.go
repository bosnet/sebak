package sync

import (
	"context"
	"encoding/json"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"

	"github.com/inconshreveable/log15"
)

//TODO(anarcher) another name is Finisher

type BlockValidator struct {
	network network.Network
	storage *storage.LevelDBBackend

	logger log15.Logger
}

type BlockValidatorOption func(*BlockValidator)

func NewBlockValidator(nw network.Network, ldb *storage.LevelDBBackend, opts ...BlockValidatorOption) *BlockValidator {
	v := &BlockValidator{
		network: nw,
		storage: ldb,

		logger: NopLogger(),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

func (v *BlockValidator) Validate(ctx context.Context, syncInfo *SyncInfo) error {
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
	//TODO: validate
	return nil
}

func (v *BlockValidator) finishBlock(ctx context.Context, syncInfo *SyncInfo) error {
	//TODO(anarcher): using leveldb.Tx or leveldb.Batch?
	ts, err := v.storage.OpenTransaction()
	if err != nil {
		return err
	}

	height := syncInfo.BlockHeight
	exists, err := block.ExistsBlockByHeight(ts, height)
	if err != nil {
		return err
	}
	if exists == true {
		v.logger.Info("This block exists", "height", height)
		return nil
	}

	blk := *syncInfo.Block
	if err := blk.Save(ts); err != nil {
		if err == errors.ErrorBlockAlreadyExists {
			return nil
		}
		return err
	}

	for _, bt := range syncInfo.Txs {
		if err := bt.Save(ts); err != nil {
			ts.Discard()
			return err
		}

		var tx *transaction.Transaction
		if err := json.Unmarshal(bt.Message, tx); err != nil {
			return err
		}

		for _, op := range tx.B.Operations {
			if err := runner.FinishOperation(ts, *tx, op, v.logger); err != nil {
				return err
			}
		}

		baSource, err := block.GetBlockAccount(ts, tx.B.Source)
		if err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			ts.Discard()
			return err
		}

		if err := baSource.Withdraw(tx.TotalAmount(true)); err != nil {
			ts.Discard()
			return err
		}

		if err := baSource.Save(ts); err != nil {
			ts.Discard()
			return err
		}
	}

	if err := ts.Commit(); err != nil {
		ts.Discard()
		return err
	}

	v.logger.Info("Finish to sync block", "height", syncInfo.BlockHeight)
	return nil
}
