package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
)

type TestSavingBlockOperationHelper struct {
	st *storage.LevelDBBackend
}

func (p *TestSavingBlockOperationHelper) Prepare() {
	p.st = block.InitTestBlockchain()
}

func (p *TestSavingBlockOperationHelper) Done() {
	p.st.Close()
}

func (p *TestSavingBlockOperationHelper) makeBlock(prevBlock block.Block, numTxs int) block.Block {
	kp := keypair.Random()

	var txs []transaction.Transaction
	var txHashes []string
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(networkID, 1, kp)
		block.SaveTransactionPool(p.st, tx)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	blk := block.TestMakeNewBlockWithPrevBlock(prevBlock, txHashes)

	blt := ballot.NewBallot(
		blk.Proposer,
		blk.Proposer,
		voting.Basis{
			BlockHash: prevBlock.Hash,
			Height:    prevBlock.Height,
			TotalTxs:  prevBlock.TotalTxs,
			TotalOps:  prevBlock.TotalOps,
		},
		txHashes,
	)
	opi, _ := ballot.NewInflationFromBallot(*blt, block.CommonKP.Address(), common.BaseReserve)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, block.CommonKP.Address(), txs...)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.ProposedTime, ptx.Transaction)
	bt.MustSave(p.st)
	block.SaveTransactionPool(p.st, ptx.Transaction)

	blk.ProposerTransaction = ptx.GetHash()

	blk.MustSave(p.st)

	for _, tx := range txs {
		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.ProposedTime, tx)
		bt.MustSave(p.st)
	}

	return blk
}

func TestSavingBlockOperation(t *testing.T) {
	p := &TestSavingBlockOperationHelper{}
	p.Prepare()
	defer p.Done()

	blk := p.makeBlock(block.GetGenesis(p.st), 5)

	for _, txHash := range blk.Transactions {
		bt, err := block.GetBlockTransaction(p.st, txHash)
		require.NoError(t, err)
		for _, opHash := range bt.Operations {
			exists, err := block.ExistsBlockOperation(p.st, opHash)
			require.NoError(t, err)
			require.False(t, exists)
		}
	}

	// with `SavingBlockOperations`
	sb := NewSavingBlockOperations(p.st, nil)
	err := sb.Check()
	require.NoError(t, err)

	// check `BlockOperation`s
	for _, txHash := range blk.Transactions {
		bt, err := block.GetBlockTransaction(p.st, txHash)
		require.NoError(t, err)
		for _, opHash := range bt.Operations {
			exists, err := block.ExistsBlockOperation(p.st, opHash)
			require.NoError(t, err)
			require.True(t, exists)
			bo, err := block.GetBlockOperation(p.st, opHash)
			require.NoError(t, err)
			require.Equal(t, blk.Height, bo.Height)
		}
	}
}

func TestSavingBlockOperationMissingInBlock(t *testing.T) {
	p := &TestSavingBlockOperationHelper{}
	p.Prepare()
	defer p.Done()

	sb := NewSavingBlockOperations(p.st, nil)

	blk0 := p.makeBlock(block.GetGenesis(p.st), 5)
	err := sb.CheckByBlock(p.st, blk0)
	require.NoError(t, err)

	// blk1 will be not save it's `BlockOperation`s
	blk1 := p.makeBlock(blk0, 5)

	blk2 := p.makeBlock(blk1, 5)
	err = sb.CheckByBlock(p.st, blk2)
	require.NoError(t, err)

	// check `BlockOperation`s in `blk1`
	for _, txHash := range blk1.Transactions {
		bt, err := block.GetBlockTransaction(p.st, txHash)
		require.NoError(t, err)
		for _, opHash := range bt.Operations {
			exists, err := block.ExistsBlockOperation(p.st, opHash)
			require.NoError(t, err)
			require.False(t, exists)
		}
	}

	err = sb.Check()
	require.NoError(t, err)

	// check `BlockOperation`s
	for _, txHash := range blk1.Transactions {
		bt, err := block.GetBlockTransaction(p.st, txHash)
		require.NoError(t, err)
		for _, opHash := range bt.Operations {
			exists, err := block.ExistsBlockOperation(p.st, opHash)
			require.NoError(t, err)
			require.True(t, exists)
		}
	}
}

func TestSavingBlockOperationWithEmptyTransactions(t *testing.T) {
	p := &TestSavingBlockOperationHelper{}
	p.Prepare()
	defer p.Done()

	// create the block, which has empty transactions.
	blk := p.makeBlock(block.GetGenesis(p.st), 0)
	require.Equal(t, len(blk.Transactions), 0)

	sb := NewSavingBlockOperations(p.st, nil)
	require.NoError(t, sb.Check())

	bt, err := block.GetBlockTransaction(p.st, blk.ProposerTransaction)
	require.NoError(t, err)
	for _, opHash := range bt.Operations {
		exists, err := block.ExistsBlockOperation(p.st, opHash)
		require.NoError(t, err)
		require.True(t, exists)
	}
}
