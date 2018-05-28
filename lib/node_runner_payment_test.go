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

func TestNodeRunnerPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kpSource, _ := keypair.Random()
	kpTarget, _ := keypair.Random()

	var accountSource, accountTarget *BlockAccount
	for _, nr := range nodeRunners {
		{
			address := kpSource.Address()
			balance := strconv.FormatInt(int64(2000), 10)
			hashed := sebakcommon.MustMakeObjectHash("")
			checkpoint := base58.Encode(hashed)

			accountSource = NewBlockAccount(address, balance, checkpoint)
			accountSource.Save(nr.Storage())
		}

		{
			balance := strconv.FormatInt(int64(2000), 10)
			hashed := sebakcommon.MustMakeObjectHash("")
			checkpoint := base58.Encode(hashed)

			accountTarget = NewBlockAccount(kpTarget.Address(), balance, checkpoint)
			accountTarget.Save(nr.Storage())
		}
	}

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

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
		nr.SetHandleBallotCheckerFuncs(ctx)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	amount := uint64(100)
	tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
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

	expectedTargetAmount, _ := accountTarget.GetBalanceAmount().Add(int64(amount))
	if baTarget.GetBalance() != int64(expectedTargetAmount) {
		t.Errorf("failed to transfer the initial amount to target; %d != %d", baTarget.GetBalance(), int64(amount))
		return
	}
	if accountSource.GetBalance()-int64(amount) != baSource.GetBalance() {
		t.Error("failed to subtract the transfered amount from source")
		return
	}
}
