package runner

import (
	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
	"github.com/stellar/go/keypair"
)

var networkID []byte = []byte("sebak-test-network")

func MakeNodeRunner() (*NodeRunner, *node.LocalNode) {
	_, n, localNode := network.CreateMemoryNetwork(nil)

	policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)

	localNode.AddValidators(localNode.ConvertToValidator())
	connectionManager := network.NewValidatorConnectionManager(
		localNode,
		n,
		policy,
	)

	conf := common.NewConfig()
	st := block.InitTestBlockchain()
	is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager, st, conf, nil)
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, policy, n, is, st, conf)
	return nodeRunner, localNode
}

func GetTransaction() (transaction.Transaction, []byte) {
	kpNewAccount, _ := keypair.Random()

	tx := transaction.MakeTransactionCreateAccount(block.GenesisKP, kpNewAccount.Address(), common.BaseReserve)
	tx.B.SequenceID = uint64(0)
	tx.Sign(block.GenesisKP, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte
	}
}

func GetCreateAccountTransaction(sequenceID uint64, amount uint64) (transaction.Transaction, []byte, *keypair.Full) {
	initialBalance := common.Amount(amount)
	kpNewAccount, _ := keypair.Random()
	tx := transaction.MakeTransactionCreateAccount(block.GenesisKP, kpNewAccount.Address(), initialBalance)
	tx.B.SequenceID = sequenceID
	tx.Sign(block.GenesisKP, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte, kpNewAccount
	}
}

func GetFreezingTransaction(kpSource *keypair.Full, sequenceID uint64, amount uint64) (transaction.Transaction, []byte, *keypair.Full) {
	initialBalance := common.Amount(amount)
	kpNewAccount, _ := keypair.Random()

	tx := transaction.MakeTransactionCreateFrozenAccount(kpSource, kpNewAccount.Address(), initialBalance, kpSource.Address())
	tx.B.SequenceID = sequenceID
	tx.Sign(kpSource, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte, kpNewAccount
	}
}

func GetUnfreezingRequestTransaction(kpSource *keypair.Full, sequenceID uint64) (transaction.Transaction, []byte) {
	tx := transaction.MakeTransactionUnfreezingRequest(kpSource)
	tx.B.SequenceID = sequenceID
	tx.Sign(kpSource, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte
	}
}

func GetUnfreezingTransaction(kpSource *keypair.Full, kpTarget *keypair.Full, sequenceID uint64, amount uint64) (transaction.Transaction, []byte) {
	unfreezingAmount := common.Amount(amount)

	tx := transaction.MakeTransactionUnfreezing(kpSource, kpTarget.Address(), unfreezingAmount)
	tx.B.SequenceID = sequenceID
	tx.Sign(kpSource, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte
	}
}

func GenerateBallot(proposer *node.LocalNode, basis voting.Basis, tx transaction.Transaction, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(sender.Address(), proposer.Address(), basis, []string{tx.GetHash()})
	b.SetVote(ballot.StateINIT, voting.YES)

	opi, _ := ballot.NewInflationFromBallot(*b, block.CommonKP.Address(), common.BaseReserve)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, block.CommonKP.Address(), tx)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)
	b.Sign(proposer.Keypair(), networkID)

	b.SetVote(ballotState, voting.YES)
	b.Sign(sender.Keypair(), networkID)

	if err := b.IsWellFormed(networkID, conf); err != nil {
		panic(err)
	}

	return b
}

func GenerateEmptyTxBallot(proposer *node.LocalNode, basis voting.Basis, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(sender.Address(), proposer.Address(), basis, []string{})
	b.SetVote(ballot.StateINIT, voting.YES)

	opi, _ := ballot.NewInflationFromBallot(*b, block.CommonKP.Address(), common.BaseReserve)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, block.CommonKP.Address())
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)
	b.Sign(proposer.Keypair(), networkID)

	b.SetVote(ballotState, voting.YES)
	b.Sign(sender.Keypair(), networkID)

	if err := b.IsWellFormed(networkID, conf); err != nil {
		panic(err)
	}

	return b
}

func ReceiveBallot(nodeRunner *NodeRunner, ballot *ballot.Ballot) error {
	data, err := ballot.Serialize()
	if err != nil {
		panic(err)
	}

	ballotMessage := common.NetworkMessage{Type: common.BallotMessage, Data: data}
	return nodeRunner.handleBallotMessage(ballotMessage)
}

func createNodeRunnerForTesting(n int, conf common.Config, recv chan struct{}) (*NodeRunner, []*node.LocalNode, *TestConnectionManager) {
	var ns []*network.MemoryNetwork
	var net *network.MemoryNetwork
	var nodes []*node.LocalNode
	for i := 0; i < n; i++ {
		_, s, v := network.CreateMemoryNetwork(net)
		net = s
		ns = append(ns, s)
		nodes = append(nodes, v)
	}

	for j := 0; j < n; j++ {
		nodes[0].AddValidators(nodes[j].ConvertToValidator())
	}

	localNode := nodes[0]
	policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)

	connectionManager := NewTestConnectionManager(
		localNode,
		ns[0],
		policy,
		recv,
	)

	st := block.InitTestBlockchain()
	is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager, st, common.NewConfig(), nil)
	is.SetProposerSelector(FixedSelector{localNode.Address()})

	nr, err := NewNodeRunner(string(networkID), localNode, policy, ns[0], is, st, conf)
	if err != nil {
		panic(err)
	}
	nr.isaacStateManager.blockTimeBuffer = 0

	return nr, nodes, connectionManager
}
