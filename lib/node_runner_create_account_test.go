package sebak

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/trie"
)

func testPrepareOneAccount() (kp *keypair.Full, balance sebakcommon.Amount, checkpoint string, nodeRunners []*NodeRunner) {
	numberOfNodes := 3
	nodeRunners = createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ = keypair.Random()

	// create new account in all nodes
	checkpoint = sebakcommon.MakeGenesisCheckpoint(networkID)
	balance = BaseFee.MustAdd(1000)
	for _, nr := range nodeRunners {
		sdb := statedb.New(nr.RootHash(), trie.NewEthDatabase(nr.Storage()))
		sdb.CreateAccount(kp.Address())
		sdb.AddBalanceWithCheckpoint(kp.Address(), balance, checkpoint)
		root, _ := sdb.CommitTrie()
		sdb.CommitDB(root)
		nr.rootHash = root
	}

	return
}

func testPrepareTwoAccount() (kp1, kp2 *keypair.Full, balance sebakcommon.Amount, checkpoint string, nodeRunners []*NodeRunner) {
	numberOfNodes := 3
	nodeRunners = createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp1, _ = keypair.Random()
	kp2, _ = keypair.Random()

	// create new account in all nodes
	checkpoint = sebakcommon.MakeGenesisCheckpoint(networkID)
	balance = BaseFee.MustAdd(100000000)
	for _, nr := range nodeRunners {
		sdb := statedb.New(nr.RootHash(), trie.NewEthDatabase(nr.Storage()))

		sdb.CreateAccount(kp1.Address())
		sdb.AddBalanceWithCheckpoint(kp1.Address(), balance, checkpoint)
		sdb.CreateAccount(kp2.Address())
		sdb.AddBalanceWithCheckpoint(kp2.Address(), balance, checkpoint)

		root, _ := sdb.CommitTrie()
		sdb.CommitDB(root)
		nr.rootHash = root
	}

	return
}

func TestNodeRunnerCreateAccount(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	kp, balance, checkpoint, nodeRunners := testPrepareOneAccount()

	kpNewAccount, _ := keypair.Random()
	initialBalance := sebakcommon.Amount(100)
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

	sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))

	if sebakcommon.MustAmountFromString(sdb.GetBalance(kpNewAccount.Address())) != initialBalance {
		t.Error("failed to transfer the initial amount to target")
		return
	}
	if balance.MustSub(tx.TotalAmount(true)) != sebakcommon.MustAmountFromString(sdb.GetBalance(kp.Address())) {
		t.Error("failed to subtract the transfered amount from source")
		return
	}
}

func TestNodeRunnerCreateAccountInvalidCheckpoint(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	kp, balance, _, nodeRunners := testPrepareOneAccount()

	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes

	initialBalance := sebakcommon.Amount(100)
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
	sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
	if sdb.ExistAccount(kpNewAccount.Address()) {
		t.Error("target account must not be created")
		return
	}

	if balance != sebakcommon.MustAmountFromString(sdb.GetBalance(kp.Address())) {
		t.Error("amount was paid from source")
		return
	}
}

func TestNodeRunnerCreateAccountSufficient(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	kp, balance, checkpoint, nodeRunners := testPrepareOneAccount()
	kpNewAccount, _ := keypair.Random()

	initialBalance := balance.MustSub(BaseFee)
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
	sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
	if sdb.ExistAccount(kpNewAccount.Address()) == false {
		t.Error("failed to get target account")
		return
	}

	if initialBalance != sebakcommon.MustAmountFromString(sdb.GetBalance(kpNewAccount.Address())) {
		t.Error("amount was not paid to target")
		return
	}
	if balance-tx.TotalAmount(true) != sebakcommon.MustAmountFromString(sdb.GetBalance(kp.Address())) {
		t.Error("amount was paid from source", balance-tx.TotalAmount(true), sebakcommon.MustAmountFromString(sdb.GetBalance(kp.Address())))
		return
	}
}

func TestNodeRunnerCreateAccountInsufficient(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	kp, balance, checkpoint, nodeRunners := testPrepareOneAccount()
	kpNewAccount, _ := keypair.Random()

	initialBalance := balance
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
	sdb := statedb.New(nr0.RootHash(), trie.NewEthDatabase(nr0.Storage()))
	if sdb.ExistAccount(kpNewAccount.Address()) {
		t.Error("target account must not be created")
		return
	}

	if balance != sebakcommon.MustAmountFromString(sdb.GetBalance(kp.Address())) {
		t.Error("amount was paid from source")
		return
	}
}
