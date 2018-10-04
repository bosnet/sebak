package runner

import (
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
)

func TestOnlyValidTransactionInTransactionPool(t *testing.T) {
	nodeRunners, rootKP := createTestNodeRunnersHTTP2NetworkWithReady(3)
	defer func() {
		for _, nr := range nodeRunners {
			nr.Stop()
		}
	}()

	nodeRunner := nodeRunners[0]

	rootAccount, _ := block.GetBlockAccount(nodeRunner.Storage(), rootKP.Address())

	TestMakeBlockAccount := func(balance common.Amount) (account *block.BlockAccount, kp *keypair.Full) {
		kp, _ = keypair.Random()
		account = block.NewBlockAccount(kp.Address(), balance)

		return
	}

	runChecker := func(tx transaction.Transaction, expectedError error) {
		messageData, _ := tx.Serialize()

		checker := &MessageChecker{
			DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleTransactionCheckerFuncs},
			NodeRunner:     nodeRunner,
			LocalNode:      nodeRunner.Node(),
			NetworkID:      networkID,
			Message:        common.NetworkMessage{Type: "message", Data: messageData},
		}

		if err := common.RunChecker(checker, nil); err != nil {
			if _, ok := err.(common.CheckerErrorStop); !ok && expectedError != nil {
				require.Error(t, err, expectedError)
			}
		}
	}

	{ // valid transaction
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000) /* 100,00000 BOS */)
		targetAccount.Save(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.SequenceID = rootAccount.SequenceID
		tx.Sign(rootKP, networkID)

		runChecker(tx, nil)

		require.True(t, nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()), "valid transaction must be in `Pool`")
	}

	{ // invalid transaction: same source already in Pool
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.SequenceID = rootAccount.SequenceID
		tx.Sign(rootKP, networkID)

		runChecker(tx, errors.ErrorTransactionSameSource)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `Pool`: same source already in `Pool`",
		)
	}

	{ // invalid transaction: source account does not exists
		_, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)

		runChecker(tx, errors.ErrorBlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `Pool`: source account does not exists",
		)
	}

	{ // invalid transaction: target account does not exists
		sourceAccount, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		_, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		sourceAccount.Save(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)
		tx.B.SequenceID = sourceAccount.SequenceID
		tx.Sign(sourceKP, networkID)

		runChecker(tx, errors.ErrorBlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must be in `Pool`: target account does not exists",
		)
	}
}

type getMissingTransactionTesting struct {
	nodeRunners   []*NodeRunner
	proposerNR    *NodeRunner
	consensusNR   *NodeRunner
	genesisBlock  block.Block
	commonAccount *block.BlockAccount
}

func (g *getMissingTransactionTesting) Prepare() {
	g.nodeRunners, _ = createTestNodeRunnersHTTP2Network(2)

	g.proposerNR = g.nodeRunners[0]

	for _, nr := range g.nodeRunners {
		nr.Ready()
		go nr.ConnectValidators()
		go func(n *NodeRunner) {
			panic(n.Network().Start())
		}(nr)
	}

	T := time.NewTicker(100 * time.Millisecond)
	stopTimer := make(chan bool)

	go func() {
		time.Sleep(5 * time.Second)
		stopTimer <- true
	}()

	go func() {
		for _ = range T.C {
			var notyet bool
			for _, nr := range g.nodeRunners {
				if nr.ConnectionManager().CountConnected() != 1 {
					notyet = true
					break
				}
			}
			if notyet {
				continue
			}
			stopTimer <- true
		}
	}()
	select {
	case <-stopTimer:
		T.Stop()
	}

	for _, nr := range g.nodeRunners {
		nr.Consensus().SetProposerSelector(FixedSelector{g.proposerNR.Node().Address()})
	}

	g.consensusNR = g.nodeRunners[1]

	g.genesisBlock, _ = block.GetBlockByHeight(g.proposerNR.Storage(), 1)
	g.commonAccount, _ = GetCommonAccount(g.proposerNR.Storage())
}

func (g *getMissingTransactionTesting) MakeBallot(numberOfTxs int) (blt *ballot.Ballot) {
	rd := round.Round{
		Number:      0,
		BlockHeight: g.genesisBlock.Height,
		BlockHash:   g.genesisBlock.Hash,
		TotalTxs:    g.genesisBlock.TotalTxs,
	}

	keys := map[string]*keypair.Full{}
	var txHashes []string
	var txs []transaction.Transaction
	for i := 0; i < numberOfTxs; i++ {
		kpA, _ := keypair.Random()
		accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve)*2)
		accountA.Save(g.proposerNR.Storage())

		kpB, _ := keypair.Random()

		tx := transaction.MakeTransactionCreateAccount(kpA, kpB.Address(), common.BaseReserve)
		tx.B.SequenceID = accountA.SequenceID
		tx.Sign(kpA, networkID)

		keys[kpA.Address()] = kpA
		txHashes = append(txHashes, tx.GetHash())
		txs = append(txs, tx)

		// inject txs to `TransactionPool`
		g.proposerNR.Consensus().TransactionPool.Add(tx)
	}

	blt = ballot.NewBallot(g.proposerNR.Node().Address(), rd, txHashes)

	ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, g.commonAccount.Address, txs...)
	blt.SetProposerTransaction(ptx)
	blt.SetVote(ballot.StateINIT, ballot.VotingYES)
	blt.Sign(g.proposerNR.Node().Keypair(), networkID)

	return blt
}

// TestGetMissingTransactionAllMissing assumes,
// * 2 NodeRunners
// * first NodeRunner proposes Ballot
// * 2nd NodeRunner does not have the transactions of the proposed Ballot
// TestGetMissingTransactionAllMissing checks 2nd NodeRunner must have all the
// transactions after `BallotTransactionChecker`.
func TestGetMissingTransactionAllMissing(t *testing.T) {
	g := &getMissingTransactionTesting{}
	g.Prepare()

	blt := g.MakeBallot(3)

	var ballotMessage common.NetworkMessage
	{
		b, _ := blt.Serialize()
		ballotMessage = common.NetworkMessage{
			Type: common.BallotMessage,
			Data: b,
		}
	}

	baseChecker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleBaseBallotCheckerFuncs},
		NodeRunner:     g.consensusNR,
		LocalNode:      g.consensusNR.Node(),
		NetworkID:      g.consensusNR.NetworkID(),
		Message:        ballotMessage,
		Log:            g.consensusNR.Log(),
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.Nil(t, err)

	var checkerFuncs = []common.CheckerFunc{
		BallotAlreadyVoted,
		BallotVote,
		BallotIsSameProposer,
		BallotGetMissingTransaction,
	}

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: checkerFuncs},
		NodeRunner:     baseChecker.NodeRunner,
		LocalNode:      baseChecker.LocalNode,
		NetworkID:      baseChecker.NetworkID,
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            baseChecker.Log,
	}

	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Nil(t, err)

	// check consensus node runner has all missing transactions.
	for _, hash := range blt.Transactions() {
		require.True(t, g.consensusNR.Consensus().TransactionPool.Has(hash))
	}
}

// TestGetMissingTransactionProposerAlsoMissing assumes,
// * 2 NodeRunners
// * first NodeRunner proposes Ballot
// * 2nd NodeRunner does not have the transactions of the proposed Ballot
// * remove one transaction from first NodeRunner
// * 2nd NodeRunner fail to get that transaction from proposer
func TestGetMissingTransactionProposerAlsoMissing(t *testing.T) {
	g := &getMissingTransactionTesting{}
	g.Prepare()

	blt := g.MakeBallot(3)

	// remove 1st tx from `TransactionPool` of proposer NodeRunner
	removedHash := blt.Transactions()[0]
	g.proposerNR.Consensus().TransactionPool.Remove(removedHash)

	var ballotMessage common.NetworkMessage
	{
		b, _ := blt.Serialize()
		ballotMessage = common.NetworkMessage{
			Type: common.BallotMessage,
			Data: b,
		}
	}

	baseChecker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleBaseBallotCheckerFuncs},
		NodeRunner:     g.consensusNR,
		LocalNode:      g.consensusNR.Node(),
		NetworkID:      g.consensusNR.NetworkID(),
		Message:        ballotMessage,
		Log:            g.consensusNR.Log(),
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.Nil(t, err)

	var checkerFuncs = []common.CheckerFunc{
		BallotAlreadyVoted,
		BallotVote,
		BallotIsSameProposer,
		BallotGetMissingTransaction,
	}

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: checkerFuncs},
		NodeRunner:     baseChecker.NodeRunner,
		LocalNode:      baseChecker.LocalNode,
		NetworkID:      baseChecker.NetworkID,
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            baseChecker.Log,
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)

	require.Equal(t, ballot.VotingNO, checker.VotingHole)
	require.Equal(t, 0, g.consensusNR.Consensus().TransactionPool.Len())
}
