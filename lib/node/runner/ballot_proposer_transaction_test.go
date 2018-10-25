package runner

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
)

type ballotCheckerProposedTransaction struct {
	genesisBlock   block.Block
	initialBalance common.Amount
	commonAccount  *block.BlockAccount
	proposerNode   *node.LocalNode
	nr             *NodeRunner

	txs      []transaction.Transaction
	txHashes []string
	keys     map[string]*keypair.Full
}

func (p *ballotCheckerProposedTransaction) Prepare() {
	nr, localNodes, _ := createNodeRunnerForTesting(2, common.NewConfig(), nil)
	p.nr = nr

	p.genesisBlock = block.GetGenesis(nr.Storage())
	p.commonAccount, _ = GetCommonAccount(nr.Storage())
	p.initialBalance, _ = GetGenesisBalance(nr.Storage())

	p.proposerNode = localNodes[1]
	nr.Consensus().SetProposerSelector(FixedSelector{p.proposerNode.Address()})

	p.keys = map[string]*keypair.Full{}
}

func (p *ballotCheckerProposedTransaction) MakeBallot(numberOfTxs int) (blt *ballot.Ballot) {
	p.txs = []transaction.Transaction{}
	p.txHashes = []string{}
	p.keys = map[string]*keypair.Full{}

	rd := voting.Basis{
		Round:     0,
		Height:    p.genesisBlock.Height,
		BlockHash: p.genesisBlock.Hash,
		TotalTxs:  p.genesisBlock.TotalTxs,
		TotalOps:  p.genesisBlock.TotalOps,
	}

	for i := 0; i < numberOfTxs; i++ {
		kpA, _ := keypair.Random()
		accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve))
		accountA.MustSave(p.nr.Storage())

		kpB, _ := keypair.Random()

		tx := transaction.MakeTransactionCreateAccount(kpA, kpB.Address(), common.Amount(1))
		tx.B.SequenceID = accountA.SequenceID
		tx.Sign(kpA, networkID)

		p.keys[kpA.Address()] = kpA
		p.txHashes = append(p.txHashes, tx.GetHash())
		p.txs = append(p.txs, tx)

		// inject txs to `Pool`
		p.nr.TransactionPool.Add(tx)
	}

	blt = ballot.NewBallot(p.proposerNode.Address(), p.proposerNode.Address(), rd, p.txHashes)

	opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, p.commonAccount.Address, p.txs...)
	opi, _ := ballot.NewInflationFromBallot(*blt, p.commonAccount.Address, p.initialBalance)

	ptx, err := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	if err != nil {
		panic(err)
	}

	blt.SetProposerTransaction(ptx)
	blt.SetVote(ballot.StateINIT, ballot.VotingYES)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	return
}

func TestProposedTransactionWithDuplicatedOperations(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	conf := common.NewConfig()
	conf.OpsLimit = 1000

	blt := p.MakeBallot(0)
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{
		ptx := blt.ProposerTransaction()
		op := ptx.B.Operations[0]
		ptx.B.Operations = []operation.Operation{op, op}

		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.Equal(t, errors.ErrorDuplicatedOperation, err)
	}
}

func TestProposedTransactionWithoutTransactions(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)

	err := blt.IsWellFormed(networkID, common.NewConfig())
	require.NoError(t, err)

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err = common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.NoError(t, err)
	require.Equal(t, ballot.VotingYES, checker.VotingHole)
}

func TestProposedTransactionWithTransactions(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with valid `CollectTxFee.Txs` count
	blt := p.MakeBallot(3)
	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.NoError(t, err)
	}

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.NoError(t, err)
	require.Equal(t, ballot.VotingYES, checker.VotingHole)
}

// TestProposedTransactionDifferentSigning checks this rule,
// `ProposerTransaction.Source()` must be same with `Ballot.Proposer()`, it
// means, `ProposerTransaction` must be signed by same KP of ballot
func TestProposedTransactionDifferentSigning(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(3)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{ // sign different source with `Ballot.Proposer()`
		newKP, _ := keypair.Random()
		ptx := blt.ProposerTransaction()
		ptx.B.Source = newKP.Address()
		ptx.Sign(newKP, networkID)
		blt.SetProposerTransaction(ptx)

		require.NotEqual(t, blt.Proposer(), ptx.Source())

		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.Equal(t, errors.ErrorInvalidProposerTransaction, err)
	}
}

func TestProposedTransactionWithTransactionsButWrongTxs(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	numberOfTxs := 3
	blt := p.MakeBallot(numberOfTxs)
	opb, _ := blt.ProposerTransaction().CollectTxFee()

	// with wrong `CollectTxFee.Txs` count
	opb.Txs = uint64(numberOfTxs - 1)
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.Equal(t, errors.ErrorInvalidOperation, err)
	}
}

func TestProposedTransactionWithWrongOperationBodyCollectTxFeeBlockData(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	conf := common.NewConfig()
	{
		// with wrong `CollectTxFee.Height`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().CollectTxFee()
		opb.Height = blt.B.Proposed.VotingBasis.Height + 1
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `CollectTxFee.BlockHash`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().CollectTxFee()
		opb.BlockHash = blt.B.Proposed.VotingBasis.BlockHash + "showme"
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `CollectTxFee.TotalTxs`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().CollectTxFee()
		opb.TotalTxs = blt.B.Proposed.VotingBasis.TotalTxs + 2
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `CollectTxFee.Txs`; this will cause the
		// insufficient collected fee.
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().CollectTxFee()
		opb.Txs = uint64(len(blt.Transactions()) + 1)
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}
}

func TestProposedTransactionWithWrongOperationBodyInflationFeeBlockData(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	conf := common.NewConfig()
	{
		// with wrong `Inflation.Height`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().Inflation()
		opb.Height = blt.B.Proposed.VotingBasis.Height + 1
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[1].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `Inflation.BlockHash`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().Inflation()
		opb.BlockHash = blt.B.Proposed.VotingBasis.BlockHash + "showme"
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[1].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `Inflation.TotalTxs`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().Inflation()
		opb.TotalTxs = blt.B.Proposed.VotingBasis.TotalTxs + 2
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[1].B = opb
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
			require.NoError(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}
}

func TestProposedTransactionWithCollectTxFeeWrongAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().CollectTxFee()
	opb.Amount = opb.Amount.MustSub(1)
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.Equal(t, errors.ErrorInvalidOperation, err)
	}
}

func TestProposedTransactionWithInflationWrongAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().Inflation()
	opb.Amount = opb.Amount.MustAdd(1)
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[1].B = opb
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.NoError(t, err)
	}

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Equal(t, errors.ErrorInvalidOperation, err)
}

func TestProposedTransactionWithNotZeroFee(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	ptx := blt.ProposerTransaction()
	ptx.B.Fee = common.Amount(1)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.Equal(t, errors.ErrorInvalidFee, err)
	}
}

func TestProposedTransactionWithCollectTxFeeWrongCommonAddress(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	wrongKP, _ := keypair.Random()
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().CollectTxFee()
	opb.Target = wrongKP.Address()
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.NoError(t, err)
	}

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Equal(t, errors.ErrorInvalidOperation, err)
}

func TestProposedTransactionWithInflationWrongCommonAddress(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	wrongKP, _ := keypair.Random()
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().Inflation()
	opb.Target = wrongKP.Address()
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[1].B = opb
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.NoError(t, err)
	}

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Equal(t, errors.ErrorInvalidOperation, err)
}

func TestProposedTransactionWithBiggerTransactionFeeThanCollected(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `CollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	var txHashes []string
	p.nr.TransactionPool.Remove(p.txHashes...)
	for _, tx := range p.txs {
		tx.B.Fee = tx.B.Fee.MustAdd(1)
		kp := p.keys[tx.Source()]
		tx.Sign(kp, networkID)
		p.nr.TransactionPool.Add(tx)
		txHashes = append(txHashes, tx.GetHash())
	}
	blt.B.Proposed.Transactions = txHashes
	blt.Sign(p.proposerNode.Keypair(), networkID)

	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt, conf)
		require.NoError(t, err)
	}

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
		VotingHole:     ballot.VotingNOTYET,
	}
	err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
	require.NoError(t, err)

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		LocalNode:      p.nr.Node(),
		NetworkID:      p.nr.NetworkID(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     ballot.VotingNOTYET,
		Log:            p.nr.Log(),
	}
	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.NoError(t, err)
	require.Equal(t, ballot.VotingNO, checker.VotingHole)
}

func TestProposedTransactionStoreWithZeroAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)
	opbc, _ := blt.ProposerTransaction().CollectTxFee()
	opbi, _ := blt.ProposerTransaction().Inflation()

	previousCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	{
		_, err := finishBallot(
			p.nr.Storage(),
			*blt,
			p.nr.TransactionPool,
			p.nr.Log(),
			p.nr.Log(),
		)
		require.NoError(t, err)
	}

	afterCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	inflationAmount, err := common.CalculateInflation(p.initialBalance)
	require.NoError(t, err)

	require.Equal(t, previousCommonAccount.Balance+inflationAmount, afterCommonAccount.Balance)

	bt, err := block.GetBlockTransaction(p.nr.Storage(), blt.ProposerTransaction().GetHash())
	require.NoError(t, err)

	require.Equal(t, blt.ProposerTransaction().GetHash(), bt.Hash)
	require.Equal(t, blt.ProposerTransaction().Source(), bt.Source)

	require.Equal(t, opbc.GetAmount()+opbi.GetAmount(), bt.Amount)
	require.Equal(t, common.Amount(0), bt.Fee)
	require.Equal(t, 2, len(bt.Operations))

	var bos []block.BlockOperation
	iterFunc, closeFunc := block.GetBlockOperationsByTxHash(p.nr.Storage(), bt.Hash, nil)
	for {
		bo, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}

		bos = append(bos, bo)
	}
	closeFunc()
	require.Equal(t, 2, len(bos))

	{ // CollectTxFee
		require.Equal(t, string(operation.TypeCollectTxFee), string(bos[0].Type))

		opbFromBlockInterface, err := operation.UnmarshalBodyJSON(bos[0].Type, bos[0].Body)
		require.NoError(t, err)
		opbFromBlock := opbFromBlockInterface.(operation.CollectTxFee)

		opb, _ := blt.ProposerTransaction().CollectTxFee()
		require.Equal(t, opb.Amount, opbFromBlock.Amount)
		require.Equal(t, opb.Target, opbFromBlock.Target)
		require.Equal(t, opb.Height, opbFromBlock.Height)
		require.Equal(t, opb.BlockHash, opbFromBlock.BlockHash)
		require.Equal(t, opb.TotalTxs, opbFromBlock.TotalTxs)
		require.Equal(t, opb.Txs, opbFromBlock.Txs)
	}

	{ // Inflation
		require.Equal(t, string(operation.TypeInflation), string(bos[1].Type))

		opbFromBlockInterface, err := operation.UnmarshalBodyJSON(bos[1].Type, bos[1].Body)
		require.NoError(t, err)
		opbFromBlock := opbFromBlockInterface.(operation.Inflation)

		opb, _ := blt.ProposerTransaction().Inflation()
		require.Equal(t, opb.Amount, opbFromBlock.Amount)
		require.Equal(t, opb.Target, opbFromBlock.Target)
		require.Equal(t, opb.Height, opbFromBlock.Height)
		require.Equal(t, opb.BlockHash, opbFromBlock.BlockHash)
		require.Equal(t, opb.TotalTxs, opbFromBlock.TotalTxs)
	}
}

func TestProposedTransactionStoreWithAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().CollectTxFee()

	previousCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	{
		_, err := finishBallot(
			p.nr.Storage(),
			*blt,
			p.nr.TransactionPool,
			p.nr.Log(),
			p.nr.Log(),
		)
		require.NoError(t, err)
	}

	afterCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	inflationAmount, err := common.CalculateInflation(p.initialBalance)
	require.NoError(t, err)
	require.Equal(t, previousCommonAccount.Balance+opb.Amount+inflationAmount, afterCommonAccount.Balance)
}

func TestProposedTransactionWithNormalOperations(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)
	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{ // with create-account operation
		ptx := blt.ProposerTransaction()
		op := ptx.B.Operations[1]

		kp, _ := keypair.Random()
		opb := operation.NewCreateAccount(kp.Address(), common.Amount(1), "")
		newOp, _ := operation.NewOperation(opb)
		ptx.B.Operations = []operation.Operation{op, newOp}

		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.Equal(t, errors.ErrorInvalidProposerTransaction, err)
	}
}

func TestProposedTransactionWithWrongNumberOfOperations(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)
	conf := common.NewConfig()
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{ // more than 2
		ptx := blt.ProposerTransaction()

		kp, _ := keypair.Random()
		opb := operation.NewCreateAccount(kp.Address(), common.Amount(1), "")
		newOp, _ := operation.NewOperation(opb)
		ptx.B.Operations = append(ptx.B.Operations, newOp)

		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		err := blt.ProposerTransaction().IsWellFormed(networkID, conf)
		require.Equal(t, errors.ErrorInvalidProposerTransaction, err)
	}
}

func TestCheckInflationBlockIncrease(t *testing.T) {
	nodeRunners, _ := createTestNodeRunnersHTTP2NetworkWithReady(1)
	defer func() {
		for _, nr := range nodeRunners {
			nr.Stop()
		}
	}()

	nr := nodeRunners[0]

	nr.ISAACStateManager().Conf.BlockTime = 0
	validators := nr.ConnectionManager().AllValidators()
	require.Equal(t, 1, len(validators))
	require.Equal(t, nr.localNode.Address(), validators[0])

	isaac := nr.Consensus()

	getCommonAccountBalance := func() common.Amount {
		commonAccount, _ := block.GetBlockAccount(nr.Storage(), nr.CommonAccountAddress)
		return commonAccount.Balance
	}

	require.Equal(t, common.Amount(0), getCommonAccountBalance())

	recv := make(chan struct{})
	nr.ISAACStateManager().SetTransitSignal(func() {
		recv <- struct{}{}
	})

	checkInflation := func(previous, inflationAmount common.Amount, blockHeight uint64) common.Amount {
		t.Logf(
			"> check inflation: block-height: %d previous: %d inflation: %d",
			blockHeight,
			previous,
			inflationAmount,
		)
		<-recv // ballot.StateINIT
		require.Equal(t, blockHeight, nr.ISAACStateManager().State().Height)
		<-recv // ballot.StateSIGN
		<-recv // ballot.StateACCEPT
		<-recv
		require.Equal(t, ballot.StateALLCONFIRM, nr.ISAACStateManager().State().BallotState)
		require.Equal(t, blockHeight+1, isaac.LatestBlock().Height)
		require.Equal(t, blockHeight, nr.ISAACStateManager().State().Height)

		expected := previous + inflationAmount
		t.Logf(
			"< inflation raised: block-height: %d previous(%d)+inflation(%d) == expected(%d) == in db: %s",
			blockHeight,
			previous,
			inflationAmount,
			expected,
			getCommonAccountBalance(),
		)
		require.Equal(t, expected, getCommonAccountBalance())

		return expected
	}

	t.Logf(
		"CalculateInflation(initial balance, inflation ratio): initial balance=%v inflation ratio=%s",
		nr.InitialBalance,
		common.InflationRatioString,
	)

	inflationAmount, err := common.CalculateInflation(nr.InitialBalance)
	require.NoError(t, err)

	var previous common.Amount
	for blockHeight := uint64(1); blockHeight < 5; blockHeight++ {
		previous = checkInflation(previous, inflationAmount, blockHeight)
	}
}

func TestProposedTransactionReachedBlockHeightEndOfInflation(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}

	p.Prepare()

	{ // Height = common.BlockHeightEndOfInflation
		genesisBlock := p.genesisBlock
		genesisBlock.Height = common.BlockHeightEndOfInflation
		genesisBlock.Hash = base58.Encode(common.MustMakeObjectHash(genesisBlock))
		p.genesisBlock = genesisBlock

		p.genesisBlock.Save(p.nr.Storage())

		blt := p.MakeBallot(4)

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
			VotingHole:     ballot.VotingNOTYET,
		}
		err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
		require.NoError(t, err)

		checker := &BallotChecker{
			DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
			NodeRunner:     p.nr,
			LocalNode:      p.nr.Node(),
			NetworkID:      p.nr.NetworkID(),
			Message:        ballotMessage,
			Ballot:         baseChecker.Ballot,
			VotingHole:     ballot.VotingNOTYET,
			Log:            p.nr.Log(),
		}
		err = common.RunChecker(checker, common.DefaultDeferFunc)
		require.NoError(t, err)
	}

	{ // Height = common.BlockHeightEndOfInflation + 1
		genesisBlock := p.genesisBlock
		genesisBlock.Height = common.BlockHeightEndOfInflation + 1
		genesisBlock.Hash = base58.Encode(common.MustMakeObjectHash(genesisBlock))
		p.genesisBlock = genesisBlock
		p.genesisBlock.Save(p.nr.Storage())

		blt := p.MakeBallot(4)

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
			VotingHole:     ballot.VotingNOTYET,
		}
		err := common.RunChecker(baseChecker, common.DefaultDeferFunc)
		require.NoError(t, err)

		checker := &BallotChecker{
			DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
			NodeRunner:     p.nr,
			LocalNode:      p.nr.Node(),
			NetworkID:      p.nr.NetworkID(),
			Message:        ballotMessage,
			Ballot:         baseChecker.Ballot,
			VotingHole:     ballot.VotingNOTYET,
			Log:            p.nr.Log(),
		}
		err = common.RunChecker(checker, common.DefaultDeferFunc)
		require.Equal(t, errors.ErrorInvalidOperation, err)
	}
}
