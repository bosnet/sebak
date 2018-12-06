package block

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
)

func TestMakeBlockAccount(kps ...*keypair.Full) *BlockAccount {
	var address string
	if len(kps) == 0 {
		address = keypair.Random().Address()
	} else {
		address = kps[0].Address()
	}
	balance := common.Amount(common.BaseReserve)

	return NewBlockAccount(address, balance)
}

var (
	GenesisKP *keypair.Full
	CommonKP  *keypair.Full
)

func init() {
	GenesisKP = keypair.Random()
	CommonKP = keypair.Random()
}

//
// Make a default-initialized test blockchain
//
// Write to the provided storage the genesis and common account,
// as well as the genesis block.
// This provide a simple, workable chain to use within tests.
//
// If anything goes wrong, `panic`
//
// Params:
//   st = Storage to write the blockchain to
//
func MakeTestBlockchain(st *storage.LevelDBBackend) {
	conf := common.NewTestConfig()
	balance := conf.InitialBalance
	genesisAccount := NewBlockAccount(GenesisKP.Address(), balance)
	if err := genesisAccount.Save(st); err != nil {
		panic(err)
	}

	commonAccount := NewBlockAccount(CommonKP.Address(), 0)
	if err := commonAccount.Save(st); err != nil {
		panic(err)
	}

	if _, err := MakeGenesisBlock(st, *genesisAccount, *commonAccount, conf.NetworkID); err != nil {
		panic(err)
	}
}

// Like `MakeTestBlockchain`, but also create a storage
func InitTestBlockchain() *storage.LevelDBBackend {
	st := storage.NewTestStorage()
	MakeTestBlockchain(st)
	return st
}

/// Version of `Block.Save` that panics on error, usable only in tests
func (b *Block) MustSave(st *storage.LevelDBBackend) {
	if err := b.Save(st); err != nil {
		panic(err)
	}
}

/// Version of `BlockAccount.Save` that panics on error, usable only in tests
func (b *BlockAccount) MustSave(st *storage.LevelDBBackend) {
	if err := b.Save(st); err != nil {
		panic(err)
	}
}

/// Version of `BlockTransaction.Save` that panics on error, usable only in tests
func (b *BlockTransaction) MustSave(st *storage.LevelDBBackend) {
	if err := b.Save(st); err != nil {
		panic(err)
	}
}

/// Version of `BlockTransaction.Save` that panics on error, usable only in tests
func (b *BlockOperation) MustSave(st *storage.LevelDBBackend) {
	if err := b.Save(st); err != nil {
		panic(err)
	}
}

func TestMakeNewBlock(transactions []string) Block {
	kp := keypair.Random()

	return *NewBlock(
		kp.Address(),
		voting.Basis{
			Height:    common.GenesisBlockHeight,
			BlockHash: "",
			TotalTxs:  uint64(len(transactions)),
			TotalOps:  uint64(len(transactions)),
		},
		"",
		transactions,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockWithPrevBlock(prevBlock Block, txs []string) Block {
	kp := keypair.Random()

	return *NewBlock(
		kp.Address(),
		voting.Basis{
			Height:    prevBlock.Height + 1,
			BlockHash: prevBlock.Hash,
			TotalTxs:  uint64(len(txs)),
			TotalOps:  uint64(len(txs)),
		},
		"",
		txs,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	for i, op := range tx.B.Operations {
		bo, err := NewBlockOperationFromOperation(op, tx, 0, i)
		if err != nil {
			panic(err)
		}
		bos = append(bos, bo)
	}

	return
}
