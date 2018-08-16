package sebak

import (
	"math/rand"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
	"boscoin.io/sebak/lib/storage"
)

var networkID []byte = []byte("sebak-test-network")

var (
	kp           *keypair.Full
	account      *block.BlockAccount
	genesisBlock Block
)

func init() {
	kp, _ = keypair.Random()
}

func createNetMemoryNetwork() (*network.MemoryNetwork, *node.LocalNode) {
	mn := network.NewMemoryNetwork()

	kp, _ := keypair.Random()
	localNode, _ := node.NewLocalNode(kp, mn.Endpoint(), "")

	mn.SetLocalNode(localNode)

	return mn, localNode
}

func MakeNodeRunner() (*NodeRunner, *node.LocalNode) {
	kp, _ := keypair.Random()

	nodeEndpoint := &common.Endpoint{Scheme: "https", Host: "https://locahost:5000"}
	localNode, _ := node.NewLocalNode(kp, nodeEndpoint, "")

	vth, _ := NewDefaultVotingThresholdPolicy(66, 66)
	is, _ := NewISAAC(networkID, localNode, vth)
	st, _ := storage.NewTestMemoryLevelDBBackend()
	network, _ := createNetMemoryNetwork()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, vth, network, is, st)
	return nodeRunner, localNode
}

func testMakeNewBlock(transactions []string) Block {
	kp, _ := keypair.Random()

	return NewBlock(
		kp.Address(),
		round.Round{
			BlockHeight: 0,
			BlockHash:   "",
		},
		transactions,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := TestMakeTransaction(networkID, n)

	block := testMakeNewBlock([]string{tx.GetHash()})
	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(block.Hash, tx, a)
}

func TestMakeOperationBodyPayment(amount int, addressList ...string) OperationBodyPayment {
	var address string
	if len(addressList) > 0 {
		address = addressList[0]
	} else {
		kp, _ := keypair.Random()
		address = kp.Address()
	}

	for amount < 0 {
		amount = rand.Intn(5000)
	}

	return OperationBodyPayment{
		Target: address,
		Amount: common.Amount(amount),
	}
}

func TestMakeOperation(amount int, addressList ...string) Operation {
	opb := TestMakeOperationBodyPayment(amount, addressList...)

	op := Operation{
		H: OperationHeader{
			Type: OperationPayment,
		},
		B: opb,
	}

	return op
}

func TestMakeTransaction(networkID []byte, n int) (kp *keypair.Full, tx Transaction) {
	kp, _ = keypair.Random()

	var ops []Operation
	for i := 0; i < n; i++ {
		ops = append(ops, TestMakeOperation(-1))
	}

	txBody := TransactionBody{
		Source:     kp.Address(),
		Fee:        BaseFee,
		SequenceID: 0,
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func TestGenerateNewSequenceID() uint64 {
	return 0
}

func TestMakeTransactionWithKeypair(networkID []byte, n int, srcKp *keypair.Full, targetKps ...*keypair.Full) (tx Transaction) {
	var ops []Operation
	var targetAddr string

	if len(targetKps) > 0 {
		targetAddr = targetKps[0].Address()
	}

	for i := 0; i < n; i++ {
		ops = append(ops, TestMakeOperation(-1, targetAddr))
	}

	tx, _ = NewTransaction(
		srcKp.Address(),
		TestGenerateNewSequenceID(),
		ops...,
	)
	tx.Sign(srcKp, networkID)

	return
}

type SelfProposerCalculator struct {
}

func (c SelfProposerCalculator) Calculate(nr *NodeRunner, _ uint64, _ uint64) string {
	return nr.localNode.Address()
}

type TheOtherProposerCalculator struct {
}

func (c TheOtherProposerCalculator) Calculate(nr *NodeRunner, _ uint64, _ uint64) string {
	for _, v := range nr.ConnectionManager().AllValidators() {
		if v != nr.localNode.Address() {
			return v
		}
	}
	panic("There is no the other validators")
}

type SelfProposerThenNotProposer struct {
}

func (c *SelfProposerThenNotProposer) Calculate(nr *NodeRunner, blockHeight uint64, roundNumber uint64) string {
	if blockHeight < 2 && roundNumber == 0 {
		return nr.localNode.Address()
	} else {
		for _, v := range nr.ConnectionManager().AllValidators() {
			if v != nr.localNode.Address() {
				return v
			}
		}
		panic("There is no the other validators")
	}
}

func GetTransaction(t *testing.T) (tx Transaction, txByte []byte) {
	initialBalance := common.Amount(1)
	kpNewAccount, _ := keypair.Random()

	tx = makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.SequenceID = account.SequenceID
	tx.Sign(kp, networkID)

	var err error

	txByte, err = tx.Serialize()
	require.Nil(t, err)

	return
}

func makeTransactionCreateAccount(kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
	opb := NewOperationBodyCreateAccount(target, common.Amount(amount))

	op := Operation{
		H: OperationHeader{
			Type: OperationCreateAccount,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Source:     kpSource.Address(),
		Fee:        BaseFee,
		SequenceID: rand.Uint64(),
		Operations: []Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx Transaction, ballotState common.BallotState, sender *node.LocalNode) *Ballot {
	ballot := NewBallot(proposer, round, []string{tx.GetHash()})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func GenerateEmptyTxBallot(t *testing.T, proposer *node.LocalNode, round round.Round, ballotState common.BallotState, sender *node.LocalNode) *Ballot {
	ballot := NewBallot(proposer, round, []string{})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func ReceiveBallot(t *testing.T, nodeRunner *NodeRunner, ballot *Ballot) error {
	data, err := ballot.Serialize()
	require.Nil(t, err)

	ballotMessage := network.Message{Type: network.BallotMessage, Data: data}
	err = nodeRunner.handleBallotMessage(ballotMessage)
	return err
}
