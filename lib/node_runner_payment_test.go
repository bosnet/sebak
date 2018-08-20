package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/trie"
)

func TestNodeRunnerPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()
	kpSource, kpTarget, balance, checkpoint, nodeRunners := testPrepareTwoAccount()

	amount := sebakcommon.Amount(1)
	tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
	tx.B.Checkpoint = checkpoint
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

	sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
	if sdb.ExistAccount(kpSource.Address()) == false {
		t.Error("failed to get source account")
		return
	}
	if sdb.ExistAccount(kpTarget.Address()) == false {
		t.Error("failed to get target account")
		return
	}

	if sebakcommon.MustAmountFromString(sdb.GetBalance(kpTarget.Address())) != balance.MustAdd(amount) {
		t.Errorf("failed to transfer the initial amount to target; %d != %d", sebakcommon.MustAmountFromString(sdb.GetBalance(kpTarget.Address())), balance.MustAdd(amount))
		return
	}
	if sebakcommon.MustAmountFromString(sdb.GetBalance(kpSource.Address())) != balance.MustSub(tx.TotalAmount(true)) {
		t.Errorf("failed to transfer the initial amount to source; %d != %d", sebakcommon.MustAmountFromString(sdb.GetBalance(kpSource.Address())), balance.MustSub(tx.TotalAmount(true)))
		return
	}
}

func TestNodeRunnerSerializedPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	kpSource, kpTarget, _, checkpoint, nodeRunners := testPrepareTwoAccount()

	nr0 := nodeRunners[0]
	{

		sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
		sourceBalance0 := sdb.GetBalance(kpSource.Address())
		targetBalance0 := sdb.GetBalance(kpTarget.Address())

		amount := sebakcommon.Amount(1)
		tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
		tx.B.Checkpoint = checkpoint
		tx.Sign(kpSource, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get 1st consensus")
				return
			}
		}

		sdb = statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
		sourceBalance1 := sdb.GetBalance(kpSource.Address())
		targetBalance1 := sdb.GetBalance(kpTarget.Address())

		if val := sebakcommon.MustAmountFromString(sourceBalance0).MustSub(tx.TotalAmount(true)); val != sebakcommon.MustAmountFromString(sourceBalance1) {
			t.Errorf("payment failed: %d != %d", val, sebakcommon.MustAmountFromString(sourceBalance1))
			return
		}
		if val := sebakcommon.MustAmountFromString(targetBalance0).MustAdd(amount); val != sebakcommon.MustAmountFromString(targetBalance1) {
			t.Errorf("payment failed: %d != %d", val, sebakcommon.MustAmountFromString(targetBalance1))
			return
		}
	}
	{
		sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
		sourceBalance0 := sdb.GetBalance(kpSource.Address())
		targetBalance0 := sdb.GetBalance(kpTarget.Address())

		amount := sebakcommon.Amount(1)
		tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
		tx.B.Checkpoint = sdb.GetCheckPoint(kpSource.Address())
		tx.Sign(kpSource, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get 2nd consensus")
				return
			}
		}

		sdb = statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
		sourceBalance1 := sdb.GetBalance(kpSource.Address())
		targetBalance1 := sdb.GetBalance(kpTarget.Address())

		if val := sebakcommon.MustAmountFromString(sourceBalance0).MustSub(tx.TotalAmount(true)); val != sebakcommon.MustAmountFromString(sourceBalance1) {
			t.Errorf("payment failed: %d != %d", val, sebakcommon.MustAmountFromString(sourceBalance1))
			return
		}
		if val := sebakcommon.MustAmountFromString(targetBalance0).MustAdd(amount); val != sebakcommon.MustAmountFromString(targetBalance1) {
			t.Errorf("payment failed: %d != %d", val, sebakcommon.MustAmountFromString(targetBalance1))
			return
		}
	}
}
