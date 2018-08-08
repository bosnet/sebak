package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
)

func TestNodeRunnerPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kpSource, _ := keypair.Random()
	kpTarget, _ := keypair.Random()

	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	var accountSource, accountTarget *BlockAccount
	for _, nr := range nodeRunners {
		{
			address := kpSource.Address()
			balance := BaseFee.MustAdd(1)

			accountSource = NewBlockAccount(address, balance, checkpoint)
			accountSource.Save(nr.Storage())
		}

		{
			balance := Amount(2000)
			accountTarget = NewBlockAccount(kpTarget.Address(), balance, checkpoint)
			accountTarget.Save(nr.Storage())
		}
	}

	amount := Amount(1)
	tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
	tx.B.Checkpoint = accountSource.Checkpoint
	tx.Sign(kpSource, networkID)

	dones := doConsensus(nodeRunners, tx)
	for _, done := range dones {
		if done.State != sebakcommon.BallotStateALLCONFIRM {
			t.Error("failed to get consensus")
			return
		}
		if done.MessageHash != tx.GetHash() {
			t.Error("failed to get consensus; found invalid message")
			return
		}
	}

	nr0 := nodeRunners[0]

	// check balance
	baSource, err := GetBlockAccount(nr0.Storage(), kpSource.Address())
	if err != nil {
		t.Error("failed to get source account")
		return
	}
	baTarget, err := GetBlockAccount(nr0.Storage(), kpTarget.Address())
	if err != nil {
		t.Error("failed to get target account")
		return
	}

	expectedTargetAmount := accountTarget.GetBalance().MustAdd(amount)
	if baTarget.GetBalance() != expectedTargetAmount {
		t.Errorf("failed to transfer the initial amount to target; %d != %d", baTarget.GetBalance(), amount)
		return
	}
	if accountSource.GetBalance().MustSub(tx.TotalAmount(true)) != baSource.GetBalance() {
		t.Error("failed to subtract the transfered amount from source")
		return
	}
}

func TestNodeRunnerSerializedPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	sourceKP, _ := keypair.Random()
	targetKP, _ := keypair.Random()

	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	var sourceAccount, targetAccount *BlockAccount
	for _, nr := range nodeRunners {
		balance := BaseFee.MustAdd(1).MustAdd(BaseFee.MustAdd(1))

		sourceAccount = NewBlockAccount(sourceKP.Address(), balance, checkpoint)
		sourceAccount.Save(nr.Storage())

		targetAccount = NewBlockAccount(targetKP.Address(), balance, checkpoint)
		targetAccount.Save(nr.Storage())
	}

	nr0 := nodeRunners[0]
	{
		sourceAccount0, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount0, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		tx := makeTransactionPayment(sourceKP, targetKP.Address(), Amount(1))
		tx.B.Checkpoint = checkpoint
		tx.Sign(sourceKP, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get 1st consensus")
				return
			}
		}

		sourceAccount1, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount1, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		if val := sourceAccount0.GetBalance().MustSub(tx.TotalAmount(true)); val != sourceAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", val, sourceAccount1.GetBalance())
			return
		}
		if val := targetAccount0.GetBalance().MustAdd(tx.B.Operations[0].B.GetAmount()); val != targetAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", val, targetAccount1.GetBalance())
			return
		}
	}

	{
		sourceAccount0, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount0, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())
		tx := makeTransactionPayment(sourceKP, targetKP.Address(), Amount(1))
		tx.B.Checkpoint = sourceAccount0.Checkpoint
		tx.Sign(sourceKP, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get 2nd consensus")
				return
			}
		}

		sourceAccount1, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount1, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		if val := sourceAccount0.GetBalance().MustSub(tx.TotalAmount(true)); val != sourceAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", val, sourceAccount1.GetBalance())
			return
		}
		if val := targetAccount0.GetBalance().MustAdd(tx.B.Operations[0].B.GetAmount()); val != targetAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", val, targetAccount1.GetBalance())
			return
		}
	}
}
