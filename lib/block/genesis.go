//
// Defines primitives used to build and interact with both the genesis
// and common budget blocks / accounts, which are special blocks
// in the BOScoin blockchain
//
package block

import (
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
)

// Returns: Genesis block
func GetGenesis(st *storage.LevelDBBackend) Block {
	if blk, err := GetBlockByHeight(st, common.GenesisBlockHeight); err != nil {
		panic(err)
	} else {
		return blk
	}
}

// MakeGenesisBlock makes genesis block.
//
// This special block has different part from the other Block
// * `Block.Proposer` is empty
// * `Block.Transaction` is empty
// * `Block.Confirmed` is `common.GenesisBlockConfirmedTime`
// * has only one `Transaction`
//
// This Transaction is different from other normal Transaction;
// * signed by `keypair.Master(string(networkID))`
// * must have only two `Operation`, `CreateAccount`
// * The first `Operation` is for genesis account
//   * `CreateAccount.Amount` is same with balance of genesis account
//   * `CreateAccount.Target` is genesis account
// * The next `Operation` is for common account
//   * `CreateAccount.Amount` is 0
//   * `CreateAccount.Target` is common account
// * `Transaction.B.Fee` is 0
func MakeGenesisBlock(st *storage.LevelDBBackend, genesisAccount BlockAccount, commonAccount BlockAccount, networkID []byte) (blk *Block, err error) {
	if genesisAccount.Address == commonAccount.Address {
		err = fmt.Errorf("genesis account and common account are same.")
		return
	}

	var exists bool
	if exists, err = ExistsBlockByHeight(st, 1); exists || err != nil {
		if exists {
			err = errors.BlockAlreadyExists
		}

		return
	}

	// create create-account transaction.
	var ops []operation.Operation
	{
		opb := operation.NewCreateAccount(genesisAccount.Address, genesisAccount.Balance, "")
		op := operation.Operation{
			H: operation.Header{
				Type: operation.TypeCreateAccount,
			},
			B: opb,
		}
		ops = append(ops, op)
	}

	{
		opb := operation.NewCreateAccount(commonAccount.Address, commonAccount.Balance, "")
		op := operation.Operation{
			H: operation.Header{
				Type: operation.TypeCreateAccount,
			},
			B: opb,
		}
		ops = append(ops, op)
	}

	txBody := transaction.Body{
		Source:     genesisAccount.Address,
		Fee:        0,
		SequenceID: genesisAccount.SequenceID,
		Operations: ops,
	}

	tx := transaction.Transaction{
		H: transaction.Header{
			Version: common.TransactionVersionV1V1,
			Created: common.GenesisBlockConfirmedTime,
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	kp := keypair.Master(string(networkID))
	tx.Sign(kp, []byte(networkID))

	blk = NewBlock(
		"",
		voting.Basis{
			Height:   common.GenesisBlockHeight,
			TotalTxs: 1,
			TotalOps: uint64(len(tx.B.Operations)), // op for creating genesis account and common account operations
		},
		"",
		[]string{tx.GetHash()},
		common.GenesisBlockConfirmedTime,
	)
	if err = blk.Save(st); err != nil {
		return
	}

	bt := NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, tx)
	if err = bt.Save(st); err != nil {
		return
	}

	if _, err = SaveTransactionPool(st, tx); err != nil {
		return
	}
	if err = bt.SaveBlockOperations(st); err != nil {
		return
	}

	return
}
