/*
	In this file, there are unittests assume that one node receive a message from validators,
	and how the state of the node changes.
*/

package runner

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"github.com/stretchr/testify/require"
)

/*
TestUnfreezingSimulation indicates the following:
	1. There are 3 nodes.
	2. A transaction for creating `A1` is confirmed.
	3. A transaction for creating `A2` is confirmed.
	4. A transaction for payment `A1` -> `A2` is confirmed.
	5. A transaction(named `tx1`) from a `A1` is pushed into tx pool.
	6. Another transaction(named `tx2`) from `A2` is proposed and confirmed.
	7. The transaction `tx1` is not removed because it is still valid.
	6. Another transaction(named `tx3`) from `A1` is proposed and confirmed.
	7. The transaction `tx1` is removed because it is not valid anymore.
*/
func TestModifiedSourceValidate(t *testing.T) {
	nr, nodes, _ := createNodeRunnerForTesting(3, common.NewTestConfig(), nil)

	st := nr.storage

	proposer := nr.localNode

	// Generate create-account transaction
	createA1Tx, _, kpNewAccount1 := GetCreateAccountTransaction(uint64(0), uint64(500000000000))

	b1, _ := MakeConsensusAndBlock(t, createA1Tx, nr, nodes, proposer)
	require.Equal(t, b1.Height, uint64(2))

	// Generate create-account transaction
	createA2Tx, _, kpNewAccount2 := GetCreateAccountTransaction(uint64(1), uint64(500000000000))

	b2, _ := MakeConsensusAndBlock(t, createA2Tx, nr, nodes, proposer)
	require.Equal(t, b2.Height, uint64(3))

	// Generate payment A1 -> A2 transaction
	paymentA1A2Tx, _ := GetPaymentTransaction(kpNewAccount1, kpNewAccount2.Address(), uint64(0), uint64(100000000000))

	b3, _ := MakeConsensusAndBlock(t, paymentA1A2Tx, nr, nodes, proposer)
	ba, _ := block.GetBlockAccount(st, kpNewAccount2.Address())

	require.Equal(t, b3.Height, uint64(4))
	require.Equal(t, uint64(ba.Balance), uint64(600000000000))

	tx1, _ := GetPaymentTransaction(kpNewAccount1, kpNewAccount2.Address(), uint64(1), uint64(100000000000))

	err := ValidateTx(nr.Storage(), common.Config{}, tx1)
	require.NoError(t, err)

	nr.TransactionPool.Add(tx1)
	require.Equal(t, 1, nr.TransactionPool.Len())

	tx2, _ := GetPaymentTransaction(kpNewAccount2, kpNewAccount1.Address(), uint64(1), uint64(100000000000))

	b4, _ := MakeConsensusAndBlock(t, tx2, nr, nodes, proposer)
	ba, _ = block.GetBlockAccount(st, kpNewAccount2.Address())

	require.Equal(t, b4.Height, uint64(5))
	require.Equal(t, uint64(ba.Balance), uint64(500000000000-common.BaseFee))

	require.NotZero(t, nr.TransactionPool.Len())

	tx3, _ := GetPaymentTransaction(kpNewAccount1, kpNewAccount2.Address(), uint64(1), uint64(100000000000))

	b5, _ := MakeConsensusAndBlock(t, tx3, nr, nodes, proposer)
	ba, _ = block.GetBlockAccount(st, kpNewAccount2.Address())

	require.Equal(t, b5.Height, uint64(6))
	require.Equal(t, uint64(ba.Balance), uint64(600000000000-common.BaseFee))

	require.Zero(t, nr.TransactionPool.Len())
}
