package block

import (
	"boscoin.io/sebak/lib/common"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func TestMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := common.Amount(2000)
	checkpoint := uuid.New().String()

	return NewBlockAccount(address, balance, checkpoint)
}
