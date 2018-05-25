package sebak

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/spikeekips/sebak/lib/util"
)

// TestNodeRunnerConsensus checks, all the validators can get consensus.
func TestNodeRunnerConsensusGetStaging(t *testing.T) {
	numberOfNodes := 10
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var handleBallotCheckerFuncs = []util.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,
		CheckNodeRunnerHandleBallotStore,
		CheckNodeRunnerHandleBallotBroadcast,
	}

	var dones []VotingStateStaging
	var deferFunc util.DeferFunc = func(n int, f util.CheckerFunc, ctx context.Context, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(util.CheckerErrorStop); ok {
			vs, _ := ctx.Value("vs").(VotingStateStaging)
			if vs.State == BallotStateALLCONFIRM {
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

	fmt.Println("currentNode", nr0.Node())

	client := nr0.TransportServer().GetClient(nr0.Node().Endpoint())

	tx := makeTransaction(nr0.Node().Keypair())
	client.SendMessage(tx)

	wg.Wait()

	for _, done := range dones {
		if done.State != BallotStateALLCONFIRM {
			t.Error("failed to get consensus")
			return
		}
		if done.MessageHash != tx.GetHash() {
			t.Error("failed to get consensus; found invalid message")
			return
		}
	}
}
