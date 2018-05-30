package sebak

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/network"
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
	checkpoint := uuid.New().String()
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := strconv.FormatInt(int64(2000), 10)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var deferFunc sebakcommon.DeferFunc = func(n int, f sebakcommon.CheckerFunc, ctx context.Context, err error) {
		if err == nil {
			return
		}

		if vs, ok := ctx.Value("vs").(VotingStateStaging); ok && vs.IsClosed() {
			if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
				dones = append(dones, vs)
				wg.Done()
			}
		}
	}

	ctx := context.WithValue(context.Background(), "deferFunc", deferFunc)
	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(ctx)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := uint64(100)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp)

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

	if baTarget.GetBalance() != int64(initialBalance) {
		t.Error("failed to transfer the initial amount to target")
		return
	}
	if account.GetBalance()-int64(initialBalance) != baSource.GetBalance() {
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
	checkpoint := uuid.New().String() // set initial checkpoint
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := strconv.FormatInt(int64(2000), 10)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var deferFunc sebakcommon.DeferFunc = func(n int, f sebakcommon.CheckerFunc, ctx context.Context, err error) {
		if err == nil {
			return
		}

		if vs, ok := ctx.Value("vs").(VotingStateStaging); ok && vs.IsClosed() {
			if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
				dones = append(dones, vs)
				wg.Done()
			}
		}
	}

	ctx := context.WithValue(context.Background(), "deferFunc", deferFunc)
	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(ctx)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := uint64(100)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)

	// set invalid checkpoint
	tx.B.Checkpoint = uuid.New().String()
	tx.Sign(kp)

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
