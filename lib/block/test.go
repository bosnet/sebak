package block

import (
	"boscoin.io/sebak/lib/common"
	"github.com/stellar/go/keypair"
)

func TestMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := common.Amount(2000)

	return NewBlockAccount(address, balance)
}
