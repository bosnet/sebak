package sebak

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/btcsuite/btcutil/base58"
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
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := strconv.FormatInt(int64(2000), 10)
		hashed := sebakcommon.MustMakeObjectHash("")
		checkpoint := base58.Encode(hashed)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,
		CheckNodeRunnerHandleBallotHistory,
		CheckNodeRunnerHandleBallotStore,
		CheckNodeRunnerHandleBallotBroadcast,
	}

	var dones []VotingStateStaging
	var deferFunc sebakcommon.DeferFunc = func(n int, f sebakcommon.CheckerFunc, ctx context.Context, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			vs, _ := ctx.Value("vs").(VotingStateStaging)
			if vs.State == sebakcommon.BallotStateALLCONFIRM {
				dones = append(dones, vs)
				wg.Done()
			}
		}
	}

	ctx := context.WithValue(context.Background(), "deferFunc", deferFunc)
	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(ctx, handleBallotCheckerFuncs...)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := uint64(100)
	tx := makeTransactionCreateAccount(kp, kpNewAccount, initialBalance)
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
