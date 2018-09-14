package block

import (
	logging "github.com/inconshreveable/log15"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-test-network")

func TestMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := common.Amount(2000)

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

	return NewBlock(
		kp.Address(),
		round.Round{
			BlockHeight: 0,
			BlockHash:   "",
		},
		transactions,
		common.NowISO8601(),
		logging.New("module", "test"),
	)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx, 0))
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	block := TestMakeNewBlock([]string{tx.GetHash()})
	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(block.Hash, block.Height, tx, a)
}
