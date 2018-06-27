package sebak

import (
	"sync"
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

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var finished []string
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.CurrentNode.Alias()); found {
			return
		}
		finished = append(finished, checker.CurrentNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc)
	}

	nr0 := nodeRunners[0]
	ep := nr0.Node().Endpoint()
	client := nr0.Network().GetClient(ep)

	initialBalance := Amount(1)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()

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

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var finished []string
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.CurrentNode.Alias()); found {
			return
		}
		finished = append(finished, checker.CurrentNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := Amount(100)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)

	// set invalid checkpoint
	tx.B.Checkpoint = uuid.New().String()
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()

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

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var finished []string
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.CurrentNode.Alias()); found {
			return
		}
		finished = append(finished, checker.CurrentNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := MustAmountFromString(account.Balance).MustSub(BaseFee)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()

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

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var finished []string
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.CurrentNode.Alias()); found {
			return
		}
		finished = append(finished, checker.CurrentNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := MustAmountFromString(account.Balance)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()

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
