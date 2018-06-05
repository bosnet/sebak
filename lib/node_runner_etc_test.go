package sebak

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/network"
)

// TestNodeRunnerLimitIncomingBallotsFromUnknownValidator checks, the incoming
// new ballot is from unknown validator and it will be ignored.
func TestNodeRunnerLimitIncomingBallotsFromUnknownValidator(t *testing.T) {
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
		balance := BaseFee.MustAdd(1)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	var wg sync.WaitGroup

	wg.Add(1)

	expectedError := sebakcommon.CheckerErrorStop{"ballot from unknown validator"}

	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			err = CheckNodeRunnerHandleBallotNotFromKnownValidators(c, args...)
			if err != nil {
				err = expectedError
			}

			return
		},
		CheckNodeRunnerHandleBallotCheckIsNew,
	}

	var ignored bool
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			return
		}

		if err.(sebakcommon.CheckerErrorStop) == expectedError {
			ignored = true
			wg.Done()
		}
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc, handleBallotCheckerFuncs...)
	}

	nr0 := nodeRunners[0]
	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := Amount(1)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)

	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

	// create new ballot with new signing

	kpUnknown, _ := keypair.Random()
	ballot, _ := NewBallotFromMessage(kpUnknown.Address(), tx)
	ballot.Sign(kpUnknown, networkID)

	client.SendBallot(ballot)

	wg.Wait()

	if !ignored {
		t.Error("unknown ballot must be ignored")
		return
	}
}
