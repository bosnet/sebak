package sync

import (
	"context"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
)

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

func (v *BlockValidator) Validate(ctx context.Context, blockInfo *BlockInfo) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		err := v.validate(ctx, blockInfo)
		if err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return v.save(ctx, blockInfo)
	}
}

func (v *BlockValidator) validate(ctx context.Context, blockInfo *BlockInfo) error {
	//TODO: validate
	return nil
}

func (v *BlockValidator) save(ctx context.Context, blockInfo *BlockInfo) error {
	//TODO(anarcher): using leveldb.Tx or leveldb.Batch?
	height := blockInfo.BlockHeight
	exists, err := block.ExistsBlockByHeight(v.storage, height)
	if err != nil {
		return err
	}
	if exists == true {
		v.logger.Info("This block exists", "height", height)
		return nil
	}

	for _, op := range blockInfo.Ops {
		if err := op.Save(v.storage); err != nil {
			if err == errors.ErrorBlockAlreadyExists {
				return nil
			}
			return err
		}
	}

	for _, tx := range blockInfo.Txs {
		if err := tx.Save(v.storage); err != nil {
			if err == errors.ErrorBlockAlreadyExists {
				return nil
			}
			return err
		}
	}

	blk := *blockInfo.Block
	if err := blk.Save(v.storage); err != nil {
		if err == errors.ErrorBlockAlreadyExists {
			return nil
		}
		return err
	}

	v.logger.Info("Save block", "height", height)
	return nil

}
