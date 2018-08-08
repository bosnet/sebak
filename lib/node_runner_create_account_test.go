package sebak

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
)

func TestNodeRunnerCreateAccount(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ := keypair.Random()
	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes
	var account *BlockAccount
	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := BaseFee.MustAdd(1)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	initialBalance := Amount(1)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp, networkID)

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
	baSource, err := GetBlockAccount(nr0.Storage(), kp.Address())
	if err != nil {
		t.Error("failed to get source account")
		return
	}
	baTarget, err := GetBlockAccount(nr0.Storage(), kpNewAccount.Address())
	if err != nil {
		t.Error("failed to get target account")
		return
	}

	if baTarget.GetBalance() != initialBalance {
		t.Error("failed to transfer the initial amount to target")
		return
	}
	if account.GetBalance().MustSub(tx.TotalAmount(true)) != baSource.GetBalance() {
		t.Error("failed to subtract the transfered amount from source")
		return
	}
}

func TestNodeRunnerCreateAccountInvalidCheckpoint(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ := keypair.Random()
	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes
	var account *BlockAccount
	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID) // set initial checkpoint
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := Amount(2000)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	initialBalance := Amount(100)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	// set invalid checkpoint
	tx.B.Checkpoint = uuid.New().String()
	tx.Sign(kp, networkID)

	dones := doConsensus(nodeRunners, tx)
	for _, done := range dones {
		if done.State != sebakcommon.BallotStateSIGN {
			t.Errorf("consensus must be failed; got invalid state, %v", done.State)
			return
		}
		if done.MessageHash != tx.GetHash() {
			t.Error("failed to get consensus; found invalid message")
			return
		}
	}

	nr0 := nodeRunners[0]

	// check balance
	_, err := GetBlockAccount(nr0.Storage(), kpNewAccount.Address())
	if err == nil {
		t.Error("target account must not be created")
		return
	}

	baSource, _ := GetBlockAccount(nr0.Storage(), kp.Address())
	if account.GetBalance() != baSource.GetBalance() {
		t.Error("amount was paid from source")
		return
	}
}

func TestNodeRunnerCreateAccountSufficient(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ := keypair.Random()
	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes
	var account *BlockAccount
	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID) // set initial checkpoint
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := BaseFee.MustAdd(1)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	initialBalance := MustAmountFromString(account.Balance).MustSub(BaseFee)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

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
	baTarget, err := GetBlockAccount(nr0.Storage(), kpNewAccount.Address())
	if err != nil {
		t.Error("failed to get target account")
		return
	}

	baSource, _ := GetBlockAccount(nr0.Storage(), kp.Address())
	if Amount(initialBalance) != Amount(baTarget.GetBalance()) {
		t.Error("amount was not paid to target")
		return
	}
	if Amount(account.GetBalance())-tx.TotalAmount(true) != Amount(baSource.GetBalance()) {
		t.Error("amount was paid from source", Amount(account.GetBalance())-tx.TotalAmount(true), Amount(baSource.GetBalance()))
		return
	}
}

func TestNodeRunnerCreateAccountInsufficient(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ := keypair.Random()
	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes
	var account *BlockAccount
	checkpoint := uuid.New().String() // set initial checkpoint
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := Amount(2000)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	initialBalance := MustAmountFromString(account.Balance)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

	dones := doConsensus(nodeRunners, tx)
	for _, done := range dones {
		if done.State != sebakcommon.BallotStateSIGN {
			t.Errorf("consensus must be failed; got invalid state, %v", done.State)
			return
		}
		if done.MessageHash != tx.GetHash() {
			t.Error("failed to get consensus; found invalid message")
			return
		}
	}

	nr0 := nodeRunners[0]

	// check balance
	_, err := GetBlockAccount(nr0.Storage(), kpNewAccount.Address())
	if err == nil {
		t.Error("target account must not be created")
		return
	}

	baSource, _ := GetBlockAccount(nr0.Storage(), kp.Address())
	if account.GetBalance() != baSource.GetBalance() {
		t.Error("amount was paid from source")
		return
	}
}
