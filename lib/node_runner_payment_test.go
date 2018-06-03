package sebak

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/network"
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

	checkpoint := uuid.New().String()
	var accountSource, accountTarget *BlockAccount
	for _, nr := range nodeRunners {
		{
			address := kpSource.Address()
			balance := strconv.FormatInt(BaseFee+1, 10)

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
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if checker.VotingStateStaging.IsEmpty() {
			return
		}

		if !checker.VotingStateStaging.IsClosed() {
			return
		}

		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	amount := uint64(1)
	tx := makeTransactionPayment(kpSource, kpTarget.Address(), amount)
	tx.B.Checkpoint = accountSource.Checkpoint
	tx.Sign(kpSource, networkID)

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
	if accountSource.GetBalance()-int64(tx.TotalAmount(true)) != baSource.GetBalance() {
		t.Error("failed to subtract the transfered amount from source")
		return
	}
}

func doConsensus(nodeRunners []*NodeRunner, tx Transaction) []VotingStateStaging {
	var wg sync.WaitGroup
	wg.Add(len(nodeRunners))

	var dones []VotingStateStaging
	var messageDeferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		return
	}

	var ballotDeferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if checker.VotingStateStaging.IsEmpty() {
			return
		}

		if !checker.VotingStateStaging.IsClosed() {
			return
		}

		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}
	for _, nr := range nodeRunners {
		nr.SetHandleMessageFromClientCheckerFuncs(messageDeferFunc)
		nr.SetHandleBallotCheckerFuncs(ballotDeferFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	client.SendMessage(tx)

	wg.Wait()

	return dones
}

func TestNodeRunnerSerializedPayment(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	sourceKP, _ := keypair.Random()
	targetKP, _ := keypair.Random()

	checkpoint := uuid.New().String()
	var sourceAccount, targetAccount *BlockAccount
	for _, nr := range nodeRunners {
		balance := (BaseFee + 1) * 2

		sourceAccount = NewBlockAccount(sourceKP.Address(), fmt.Sprintf("%d", balance), checkpoint)
		sourceAccount.Save(nr.Storage())

		targetAccount = NewBlockAccount(targetKP.Address(), fmt.Sprintf("%d", balance), checkpoint)
		targetAccount.Save(nr.Storage())
	}

	nr0 := nodeRunners[0]
	{
		sourceAccount0, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount0, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		tx := makeTransactionPayment(sourceKP, targetKP.Address(), uint64(1))
		tx.B.Checkpoint = checkpoint
		tx.Sign(sourceKP, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get consensus")
				return
			}
		}

		sourceAccount1, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount1, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		if sourceAccount0.GetBalance()-int64(tx.TotalAmount(true)) != sourceAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", sourceAccount0.GetBalance()-int64(tx.TotalAmount(true)), sourceAccount1.GetBalance())
			return
		}
		if targetAccount0.GetBalance()+int64(tx.B.Operations[0].B.GetAmount()) != targetAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", targetAccount0.GetBalance()-int64(tx.TotalAmount(true)), targetAccount1.GetBalance())
			return
		}
	}

	{
		sourceAccount0, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount0, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())
		tx := makeTransactionPayment(sourceKP, targetKP.Address(), uint64(1))
		tx.B.Checkpoint = sourceAccount0.Checkpoint
		tx.Sign(sourceKP, networkID)

		dones := doConsensus(nodeRunners, tx)
		for _, vs := range dones {
			if vs.State != sebakcommon.BallotStateALLCONFIRM {
				t.Error("failed to get consensus")
				return
			}
		}

		sourceAccount1, _ := GetBlockAccount(nr0.Storage(), sourceKP.Address())
		targetAccount1, _ := GetBlockAccount(nr0.Storage(), targetKP.Address())

		if sourceAccount0.GetBalance()-int64(tx.TotalAmount(true)) != sourceAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", sourceAccount0.GetBalance()-int64(tx.TotalAmount(true)), sourceAccount1.GetBalance())
			return
		}
		if targetAccount0.GetBalance()+int64(tx.B.Operations[0].B.GetAmount()) != targetAccount1.GetBalance() {
			t.Errorf("payment failed: %d != %d", targetAccount0.GetBalance()-int64(tx.TotalAmount(true)), targetAccount1.GetBalance())
			return
		}
	}
}
