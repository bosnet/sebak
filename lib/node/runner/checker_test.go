package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
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
		kp = keypair.Random()
		account = block.NewBlockAccount(kp.Address(), balance)

		return
	}

	runChecker := func(tx transaction.Transaction, expectedError error) {
		messageData, _ := tx.Serialize()

		checker := &MessageChecker{
			DefaultChecker:  common.DefaultChecker{Funcs: HandleTransactionCheckerFuncs},
			LocalNode:       nodeRunner.Node(),
			Consensus:       nodeRunner.Consensus(),
			TransactionPool: nodeRunner.TransactionPool,
			Storage:         nodeRunner.Storage(),
			NetworkID:       networkID,
			Message:         common.NetworkMessage{Type: "message", Data: messageData},
			Log:             nodeRunner.Log(),
			Conf:            nodeRunner.Conf,
		}

		if err := common.RunChecker(checker, nil); err != nil {
			if _, ok := err.(common.CheckerErrorStop); !ok && expectedError != nil {
				require.Error(t, err, expectedError)
			}
		}
	}

	{ // valid transaction
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000) /* 100,00000 BOS */)
		targetAccount.MustSave(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.SequenceID = rootAccount.SequenceID
		tx.Sign(rootKP, networkID)

		runChecker(tx, nil)

		require.True(t, nodeRunner.TransactionPool.Has(tx.GetHash()), "valid transaction must be in `Pool`")
	}

	{ // invalid transaction: same source already in Pool
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.MustSave(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.SequenceID = rootAccount.SequenceID
		tx.Sign(rootKP, networkID)

		runChecker(tx, errors.TransactionSameSourceInBallot)

		require.False(
			t,
			nodeRunner.TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `Pool`: same source already in `Pool`",
		)
	}

	{ // invalid transaction: source account does not exists
		_, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.MustSave(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)

		runChecker(tx, errors.BlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `Pool`: source account does not exists",
		)
	}

	{ // invalid transaction: target account does not exists
		sourceAccount, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		_, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		sourceAccount.MustSave(nodeRunner.Storage())

		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)
		tx.B.SequenceID = sourceAccount.SequenceID
		tx.Sign(sourceKP, networkID)

		runChecker(tx, errors.BlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.TransactionPool.Has(tx.GetHash()),
			"invalid transaction must be in `Pool`: target account does not exists",
		)
	}
}

type getMissingTransactionTesting struct {
	nodeRunners    []*NodeRunner
	proposerNR     *NodeRunner
	consensusNR    *NodeRunner
	genesisBlock   block.Block
	commonAccount  *block.BlockAccount
	initialBalance common.Amount
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

	g.genesisBlock = block.GetGenesis(g.proposerNR.Storage())
	g.commonAccount, _ = GetCommonAccount(g.proposerNR.Storage())
	g.initialBalance, _ = GetGenesisBalance(g.proposerNR.Storage())
}

func (g *getMissingTransactionTesting) MakeBallot(numberOfTxs int) (blt *ballot.Ballot) {
	rd := voting.Basis{
		Round:     0,
		Height:    g.genesisBlock.Height,
		BlockHash: g.genesisBlock.Hash,
		TotalTxs:  g.genesisBlock.TotalTxs,
	}

	keys := map[string]*keypair.Full{}
	var txHashes []string
	var txs []transaction.Transaction
	for i := 0; i < numberOfTxs; i++ {
		kpA := keypair.Random()
		accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve)*2)
		accountA.MustSave(g.proposerNR.Storage())
		accountA.MustSave(g.consensusNR.Storage())

		kpB := keypair.Random()

		tx := transaction.MakeTransactionCreateAccount(networkID, kpA, kpB.Address(), common.BaseReserve)
		tx.B.SequenceID = accountA.SequenceID
		tx.Sign(kpA, networkID)

		keys[kpA.Address()] = kpA
		txHashes = append(txHashes, tx.GetHash())
		txs = append(txs, tx)

		// inject txs to `TransactionPool`
		err := ValidateTx(g.proposerNR.Storage(), tx)
		if err != nil {
			panic(err)
		}
		g.proposerNR.TransactionPool.Add(tx)
		_, err = block.SaveTransactionPool(g.proposerNR.Storage(), tx)
		if err != nil {
			panic(err)
		}
	}

	blt = ballot.NewBallot(g.proposerNR.Node().Address(), g.proposerNR.Node().Address(), rd, txHashes)

	opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, g.commonAccount.Address, txs...)
	opi, _ := ballot.NewInflationFromBallot(*blt, g.commonAccount.Address, g.initialBalance)

	ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	blt.SetProposerTransaction(ptx)
	blt.SetVote(ballot.StateINIT, voting.YES)
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

	for _, hash := range blt.Transactions() {
		exists, err := block.ExistsTransactionPool(g.consensusNR.Storage(), hash)
		require.NoError(t, err)
		require.False(t, exists)
	}

	baseChecker := &BallotChecker{
		DefaultChecker:       common.DefaultChecker{Funcs: DefaultHandleBaseBallotCheckerFuncs},
		NodeRunner:           g.consensusNR,
		LocalNode:            g.consensusNR.Node(),
		NetworkID:            g.consensusNR.NetworkID(),
		Message:              ballotMessage,
		Log:                  g.consensusNR.Log(),
		VotingHole:           voting.NOTYET,
		LatestUpdatedSources: make(map[string]struct{}),
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	var checkerFuncs = []common.CheckerFunc{
		BallotAlreadyVoted,
		BallotVote,
		BallotIsSameProposer,
		BallotGetMissingTransaction,
	}

	checker := &BallotChecker{
		DefaultChecker:       common.DefaultChecker{Funcs: checkerFuncs},
		NodeRunner:           baseChecker.NodeRunner,
		LocalNode:            baseChecker.LocalNode,
		NetworkID:            baseChecker.NetworkID,
		Message:              ballotMessage,
		Ballot:               baseChecker.Ballot,
		VotingHole:           voting.NOTYET,
		Log:                  baseChecker.Log,
		LatestUpdatedSources: baseChecker.LatestUpdatedSources,
	}

	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.NoError(t, err)

	// check consensus node runner has all missing transactions.
	for _, hash := range blt.Transactions() {
		exists, err := block.ExistsTransactionPool(g.consensusNR.Storage(), hash)
		require.NoError(t, err)
		require.True(t, exists)
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
	g.proposerNR.TransactionPool.Remove(removedHash)
	block.DeleteTransactionPool(g.proposerNR.Storage(), removedHash)

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
		VotingHole:     voting.NOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

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
		VotingHole:     voting.NOTYET,
		Log:            baseChecker.Log,
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.NoError(t, err)

	require.Equal(t, voting.NO, checker.VotingHole)
	require.Equal(t, 0, g.consensusNR.TransactionPool.Len())
}

type irregularIncomingBallot struct {
	nr    *NodeRunner
	nodes []*node.LocalNode

	genesisBlock   block.Block
	commonAccount  *block.BlockAccount
	initialBalance common.Amount

	keyA     *keypair.Full
	accountA *block.BlockAccount
}

func (p *irregularIncomingBallot) prepare() {
	p.nr, p.nodes, _ = createNodeRunnerForTesting(2, common.NewConfig(), nil)

	p.genesisBlock = block.GetGenesis(p.nr.Storage())
	p.commonAccount, _ = GetCommonAccount(p.nr.Storage())
	p.initialBalance, _ = GetGenesisBalance(p.nr.Storage())

	p.nr.Consensus().SetProposerSelector(FixedSelector{p.nr.Node().Address()})
}

func (p *irregularIncomingBallot) runChecker(blt ballot.Ballot) (checker *BallotChecker, err error) {
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
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Log:            p.nr.Log(),
		VotingHole:     voting.NOTYET,
	}
	{
		err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
		if err != nil {
			panic(err)
		}
	}

	var funcs []common.CheckerFunc
	if blt.State() == ballot.StateINIT {
		funcs = DefaultHandleINITBallotCheckerFuncs
	} else if blt.State() == ballot.StateSIGN {
		funcs = DefaultHandleSIGNBallotCheckerFuncs
	}

	checker = &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: funcs},
		NodeRunner:     baseChecker.NodeRunner,
		LocalNode:      baseChecker.LocalNode,
		NetworkID:      baseChecker.NetworkID,
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     voting.NOTYET,
		Log:            baseChecker.Log,
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)

	return
}

func (p *irregularIncomingBallot) makeBallot(state ballot.State) (blt *ballot.Ballot) {
	rd := voting.Basis{
		Round:     0,
		Height:    p.genesisBlock.Height,
		BlockHash: p.genesisBlock.Hash,
		TotalTxs:  p.genesisBlock.TotalTxs,
	}

	p.keyA = keypair.Random()
	p.accountA = block.NewBlockAccount(p.keyA.Address(), common.Amount(common.BaseReserve)*2)
	p.accountA.MustSave(p.nr.Storage())

	kpB := keypair.Random()

	tx := transaction.MakeTransactionCreateAccount(networkID, p.keyA, kpB.Address(), common.BaseReserve)
	tx.B.SequenceID = p.accountA.SequenceID
	tx.Sign(p.keyA, networkID)

	// inject txs to `TransactionPool`
	p.nr.TransactionPool.Add(tx)

	blt = ballot.NewBallot(p.nr.Node().Address(), p.nr.Node().Address(), rd, []string{tx.GetHash()})

	opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, p.commonAccount.Address, tx)
	opi, _ := ballot.NewInflationFromBallot(*blt, p.commonAccount.Address, p.initialBalance)

	ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	blt.SetProposerTransaction(ptx)
	blt.SetVote(state, voting.YES)
	blt.Sign(p.nr.Node().Keypair(), networkID)

	return blt
}

// TestRegularIncomingBallots checks the normal situation of consensus; node
// receives `INIT` ballot and then, `SIGN` ballot with regular sequence.
func TestRegularIncomingBallots(t *testing.T) {
	p := &irregularIncomingBallot{}
	p.prepare()

	cm := p.nr.ConnectionManager().(*TestConnectionManager)
	require.Equal(t, 0, len(cm.Messages()))

	// send `INIT` ballot
	blt := p.makeBallot(ballot.StateINIT)
	_, err := p.runChecker(*blt)
	require.NoError(t, err)
	require.Equal(t, 1, len(cm.Messages())) // this check node broadcast the new SIGN ballot

	received := cm.Messages()[0].(ballot.Ballot)
	require.Equal(t, blt.H.ProposerSignature, received.H.ProposerSignature)

	_, err = p.runChecker(received)
	require.NoError(t, err)
	require.Equal(t, 1, len(cm.Messages())) // this check node does not broadcast
}

// TestIrregularIncomingBallots checks the normal situation of consensus; node
// receives `SIGN` ballot and then, `INIT` ballot with irregular sequence. This
// will check the node must broadcast new SIGN ballot.
func TestIrregularIncomingBallots(t *testing.T) {
	p := &irregularIncomingBallot{}
	p.prepare()

	cm := p.nr.ConnectionManager().(*TestConnectionManager)
	require.Equal(t, 0, len(cm.Messages()))

	// send `SIGN` ballot
	initBallot := p.makeBallot(ballot.StateINIT)

	signBallot := &ballot.Ballot{}
	*signBallot = *initBallot
	signBallot.SetVote(ballot.StateSIGN, voting.YES)
	signBallot.Sign(p.nodes[1].Keypair(), networkID)

	_, err := p.runChecker(*signBallot)
	require.NoError(t, err)
	require.Equal(t, 0, len(cm.Messages()))

	_, err = p.runChecker(*initBallot)
	require.NoError(t, err)
	require.Equal(t, 1, len(cm.Messages()))

	// check the broadcasted ballot is valid `SIGN` ballot
	received := cm.Messages()[0].(ballot.Ballot)
	require.Equal(t, initBallot.H.ProposerSignature, received.H.ProposerSignature)
	require.Equal(t, ballot.StateSIGN, received.State())
}
