package runner

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

type ballotCheckerProposedTransaction struct {
	genesisBlock  block.Block
	commonAccount *block.BlockAccount
	proposerNode  *node.LocalNode
	nr            *NodeRunner

	txs      []transaction.Transaction
	txHashes []string
	keys     map[string]*keypair.Full
}

func (p *ballotCheckerProposedTransaction) Prepare() {
	nr, localNodes, _ := createNodeRunnerForTesting(2, consensus.NewISAACConfiguration(), nil)
	p.nr = nr

	p.genesisBlock, _ = block.GetBlockByHeight(nr.Storage(), 1)
	p.commonAccount, _ = GetCommonAccount(nr.Storage())

	p.proposerNode = localNodes[1]
	nr.Consensus().SetProposerSelector(FixedSelector{p.proposerNode.Address()})

	p.keys = map[string]*keypair.Full{}
}

func (p *ballotCheckerProposedTransaction) MakeBallot(numberOfTxs int) (blt *ballot.Ballot) {
	p.txs = []transaction.Transaction{}
	p.txHashes = []string{}
	p.keys = map[string]*keypair.Full{}

	rd := round.Round{
		Number:      0,
		BlockHeight: p.genesisBlock.Height,
		BlockHash:   p.genesisBlock.Hash,
		TotalTxs:    p.genesisBlock.TotalTxs,
	}

	for i := 0; i < numberOfTxs; i++ {
		kpA, _ := keypair.Random()
		accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve))
		accountA.Save(p.nr.Storage())

		kpB, _ := keypair.Random()

		tx := transaction.MakeTransactionCreateAccount(kpA, kpB.Address(), common.Amount(1))
		tx.B.SequenceID = accountA.SequenceID
		tx.Sign(kpA, networkID)

		p.keys[kpA.Address()] = kpA
		p.txHashes = append(p.txHashes, tx.GetHash())
		p.txs = append(p.txs, tx)

		// inject txs to `TransactionPool`
		p.nr.Consensus().TransactionPool.Add(tx)
	}

	blt = ballot.NewBallot(p.proposerNode, rd, p.txHashes)

	ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, p.commonAccount.Address, p.txs...)
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)

	blt.SetProposerTransaction(ptx)
	blt.SetVote(ballot.StateINIT, ballot.VotingYES)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	return
}

func TestBallotCheckerProposedTransactionWithoutTransactions(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)

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
	require.Nil(t, err)

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
	require.Nil(t, err)
	require.Equal(t, ballot.VotingYES, checker.VotingHole)
}

func TestBallotCheckerProposedTransactionWithTransactions(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with valid `OperationBodyCollectTxFee.Txs` count
	blt := p.MakeBallot(3)
	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Nil(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
		require.Nil(t, err)
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
	require.Nil(t, err)

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
	require.Nil(t, err)
	require.Equal(t, ballot.VotingYES, checker.VotingHole)
}

func TestBallotCheckerProposedTransactionWithTransactionsButWrongTxs(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Txs` count
	numberOfTxs := 3
	blt := p.MakeBallot(numberOfTxs)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
	opb.Txs = uint64(numberOfTxs - 1)
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Nil(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
		require.Equal(t, errors.ErrorInvalidOperation, err)
	}
}

func TestBallotCheckerProposedTransactionWithTransactionsButWrongBlockData(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	{
		// with wrong `OperationBodyCollectTxFee.BlockHeight`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
		opb.BlockHeight = blt.B.Proposed.Round.BlockHeight + 1
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		ptx.Sign(p.proposerNode.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID)
			require.Nil(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `OperationBodyCollectTxFee.BlockHash`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
		opb.BlockHash = blt.B.Proposed.Round.BlockHash + "showme"
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		ptx.Sign(p.proposerNode.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID)
			require.Nil(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `OperationBodyCollectTxFee.TotalTxs`
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
		opb.TotalTxs = blt.B.Proposed.Round.TotalTxs + 2
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		ptx.Sign(p.proposerNode.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID)
			require.Nil(t, err)
		}
		{
			err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}

	{
		// with wrong `OperationBodyCollectTxFee.Txs`; this will cause the
		// insufficient collected fee.
		blt := p.MakeBallot(4)
		opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
		opb.Txs = uint64(len(blt.Transactions()) + 1)
		ptx := blt.ProposerTransaction()
		ptx.B.Operations[0].B = opb
		ptx.Sign(p.proposerNode.Keypair(), networkID)
		blt.SetProposerTransaction(ptx)
		blt.Sign(p.proposerNode.Keypair(), networkID)

		{
			err := blt.ProposerTransaction().IsWellFormed(networkID)
			require.Equal(t, errors.ErrorInvalidOperation, err)
		}
	}
}

func TestBallotCheckerProposedTransactionWithTransactionsButWrongAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
	opb.Amount = opb.Amount.MustSub(1)
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Equal(t, errors.ErrorInvalidOperation, err)
	}
}

func TestBallotCheckerProposedTransactionWithNotZeroFee(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	ptx := blt.ProposerTransaction()
	ptx.B.Fee = common.Amount(1)
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Equal(t, errors.ErrorInvalidFee, err)
	}
}

func TestBallotCheckerProposedTransactionWithWrongCommonAddress(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Amount` count
	wrongKP, _ := keypair.Random()
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
	opb.Target = wrongKP.Address()
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

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
	require.Equal(t, errors.ErrorInvalidOperation, err)
}

func TestBallotCheckerProposedTransactionWithBiggerTransactionFeeThanCollected(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Amount` count
	blt := p.MakeBallot(4)
	var txHashes []string
	p.nr.Consensus().TransactionPool.Remove(p.txHashes...)
	for _, tx := range p.txs {
		tx.B.Fee = tx.B.Fee.MustAdd(1)
		kp := p.keys[tx.Source()]
		tx.Sign(kp, networkID)
		p.nr.Consensus().TransactionPool.Add(tx)
		txHashes = append(txHashes, tx.GetHash())
	}
	blt.B.Proposed.Transactions = txHashes
	blt.Sign(p.proposerNode.Keypair(), networkID)

	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Nil(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
		require.Nil(t, err)
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
	require.Nil(t, err)

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
	require.Nil(t, err)
	require.Equal(t, ballot.VotingNO, checker.VotingHole)
}

func TestBallotCheckerProposedTransactionWithBadCommonAccount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	// with wrong `OperationBodyCollectTxFee.Amount` count
	wrongKP, _ := keypair.Random()
	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()
	opb.Target = wrongKP.Address()
	ptx := blt.ProposerTransaction()
	ptx.B.Operations[0].B = opb
	ptx.Sign(p.proposerNode.Keypair(), networkID)
	blt.SetProposerTransaction(ptx)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	{
		err := blt.ProposerTransaction().IsWellFormed(networkID)
		require.Nil(t, err)
	}
	{
		err := blt.ProposerTransaction().IsWellFormedWithBallot(networkID, *blt)
		require.Nil(t, err)
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
	require.Equal(t, errors.ErrorInvalidOperation, err)
}

func TestBallotCheckerProposedTransactionStoreWithZeroAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(0)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()

	previousCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	{
		_, err := finishBallot(
			p.nr.Storage(),
			*blt,
			p.nr.Consensus().TransactionPool,
			p.nr.Log(),
			p.nr.Log(),
		)
		require.Nil(t, err)
	}

	afterCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	require.Equal(t, previousCommonAccount.Balance, afterCommonAccount.Balance)

	bt, err := block.GetBlockTransaction(p.nr.Storage(), blt.ProposerTransaction().GetHash())
	require.Nil(t, err)

	require.Equal(t, blt.ProposerTransaction().GetHash(), bt.Hash)
	require.Equal(t, blt.ProposerTransaction().Source(), bt.Source)
	require.Equal(t, opb.GetAmount(), bt.Amount)
	require.Equal(t, common.Amount(0), bt.Fee)
	require.Equal(t, 1, len(bt.Operations))

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
	require.Equal(t, 1, len(bos))
	require.Equal(t, transaction.OperationCollectTxFee, bos[0].Type)

	opbFromBlockInterface, err := transaction.UnmarshalOperationBodyJSON(bos[0].Type, bos[0].Body)
	require.Nil(t, err)
	opbFromBlock := opbFromBlockInterface.(transaction.OperationBodyCollectTxFee)
	require.Equal(t, opb.Amount, opbFromBlock.Amount)
	require.Equal(t, opb.Target, opbFromBlock.Target)
	require.Equal(t, opb.BlockHeight, opbFromBlock.BlockHeight)
	require.Equal(t, opb.BlockHash, opbFromBlock.BlockHash)
	require.Equal(t, opb.TotalTxs, opbFromBlock.TotalTxs)
	require.Equal(t, opb.Txs, opbFromBlock.Txs)
}

func TestBallotCheckerProposedTransactionStoreWithAmount(t *testing.T) {
	p := &ballotCheckerProposedTransaction{}
	p.Prepare()

	blt := p.MakeBallot(4)
	opb, _ := blt.ProposerTransaction().OperationBodyCollectTxFee()

	previousCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	{
		_, err := finishBallot(
			p.nr.Storage(),
			*blt,
			p.nr.Consensus().TransactionPool,
			p.nr.Log(),
			p.nr.Log(),
		)
		require.Nil(t, err)
	}

	afterCommonAccount, _ := block.GetBlockAccount(p.nr.Storage(), p.commonAccount.Address)

	require.Equal(t, previousCommonAccount.Balance+opb.Amount, afterCommonAccount.Balance)
}
