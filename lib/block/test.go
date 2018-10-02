package block

import (
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-test-network")

func TestMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := common.Amount(common.BaseReserve)

	return NewBlockAccount(address, balance)
}

var (
	GenesisKP *keypair.Full
	CommonKP  *keypair.Full
)

func init() {
	var err error
	GenesisKP, err = keypair.Random()
	if err != nil {
		panic(err)
	}
	CommonKP, err = keypair.Random()
	if err != nil {
		panic(err)
	}
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
	balance := common.MaximumBalance
	genesisAccount := NewBlockAccount(GenesisKP.Address(), balance)
	if err := genesisAccount.Save(st); err != nil {
		panic(err)
	}

	commonAccount := NewBlockAccount(CommonKP.Address(), 0)
	if err := commonAccount.Save(st); err != nil {
		panic(err)
	}

	if _, err := MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID); err != nil {
		panic(err)
	}
}

// Like `MakeTestBlockchain`, but also create a storage
func InitTestBlockchain() *storage.LevelDBBackend {
	st := storage.NewTestStorage()
	MakeTestBlockchain(st)
	return st
}

func TestMakeNewBlock(transactions []string) Block {
	kp, _ := keypair.Random()

	return *NewBlock(
		kp.Address(),
		round.Round{
			BlockHeight: 0,
			BlockHash:   "",
		},
		"",
		transactions,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockWithPrevBlock(prevBlock Block, txs []string) Block {
	kp, _ := keypair.Random()

	return NewBlock(
		kp.Address(),
		round.Round{
			BlockHeight: prevBlock.Height,
			BlockHash:   prevBlock.Hash,
		},
		txs,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bo, err := NewBlockOperationFromOperation(op, tx, 0)
		if err != nil {
			panic(err)
		}
		bos = append(bos, bo)
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	block := TestMakeNewBlock([]string{tx.GetHash()})
	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(block.Hash, block.Height, block.Confirmed, tx, a)
}
