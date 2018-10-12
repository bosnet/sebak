package block

import (
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
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
	kp           *keypair.Full
	account      *BlockAccount
	genesisBlock Block
)

func init() {
	kp, _ = keypair.Random()
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
