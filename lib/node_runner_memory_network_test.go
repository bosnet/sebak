package sebak

import (
	"context"
	"sync"
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func createNetMemoryNetwork() (*sebaknetwork.MemoryNetwork, *sebakcommon.Validator) {
	mn := sebaknetwork.NewMemoryNetwork()

	kp, _ := keypair.Random()
	validator, _ := sebakcommon.NewValidator(kp.Address(), mn.Endpoint(), "")
	validator.SetKeypair(kp)

	mn.SetContext(context.WithValue(context.Background(), "currentNode", validator))

	return mn, validator
}

func makeTransaction(kp *keypair.Full) (tx Transaction) {
	var ops []Operation
	ops = append(ops, TestMakeOperation(-1))

	txBody := TransactionBody{
		Source:     kp.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func makeTransactionPayment(kpSource *keypair.Full, target string, amount Amount) (tx Transaction) {
	opb := NewOperationBodyPayment(target, amount)

	op := Operation{
		H: OperationHeader{
			Type: OperationPayment,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Source:     kpSource.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func makeTransactionCreateAccount(kpSource *keypair.Full, target string, amount Amount) (tx Transaction) {
	opb := NewOperationBodyCreateAccount(target, Amount(amount))

	op := Operation{
		H: OperationHeader{
			Type: OperationCreateAccount,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Source:     kpSource.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func createNodeRunners(n int) []*NodeRunner {
	var ns []*sebaknetwork.MemoryNetwork
	var validators []*sebakcommon.Validator
	for i := 0; i < n; i++ {
		s, v := createNetMemoryNetwork()
		ns = append(ns, s)
		validators = append(validators, v)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			validators[i].AddValidators(validators[j])
		}
	}

	var nodeRunners []*NodeRunner
	for i := 0; i < n; i++ {
		v := validators[i]
		p, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
		p.SetValidators(len(v.GetValidators()) + 1)
		is, _ := NewISAAC(networkID, v, p)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
		nr := NewNodeRunner(string(networkID), v, p, ns[i], is, st)
		nodeRunners = append(nodeRunners, nr)
	}

	return nodeRunners
}

func createNodeRunnersWithReady(n int) []*NodeRunner {
	nodeRunners := createNodeRunners(n)

	for _, nr := range nodeRunners {
		go nr.Start()
		//defer nr.Stop()
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
			for _, nr := range nodeRunners {
				if nr.ConnectionManager().CountConnected() != n-1 {
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

	return nodeRunners
}

// TestMemoryNetworkCreate checks, `NodeRunner` is correctly started and
// `GetNodeInfo` returns the validator information correctly.
func TestMemoryNetworkCreate(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	nodeRunners := createNodeRunners(4)
	for _, nr := range nodeRunners {
		go nr.Start()
		defer nr.Stop()
	}

	mn := nodeRunners[0].Network()
	for _, nr := range nodeRunners {
		client := mn.GetClient(nr.Node().Endpoint())
		b, err := client.GetNodeInfo()
		if err != nil {
			t.Error(err)
			return
		}

		if rv, err := sebakcommon.NewValidatorFromString(b); err != nil {
			t.Error("invalid validator data was received")
			return
		} else if !nr.Node().DeepEqual(rv) {
			t.Error("loaded validator does not match")
			return
		}
	}
}

// TestMemoryNetworkHandleMessageFromClient checks, the message can be
// broadcasted correctly in node.
func TestMemoryNetworkHandleMessageFromClient(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	nodeRunners := createNodeRunners(4)
	for _, nr := range nodeRunners {
		go nr.Start()
		defer nr.Stop()
	}

	c0 := nodeRunners[0].Network().GetClient(nodeRunners[0].Node().Endpoint())

	chanGotMessageFromClient := make(chan Ballot)
	var handleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageISAACReceiveMessage,
		CheckNodeRunnerHandleMessageSignBallot,
		func(c sebakcommon.Checker, args ...interface{}) error {
			checker := c.(*NodeRunnerHandleMessageChecker)

			checker.NodeRunner.log.Debug("ballot from client will be broadcasted", "ballot", checker.Ballot.MessageHash())
			chanGotMessageFromClient <- checker.Ballot

			return nil
		},
	}

	nodeRunners[0].SetHandleMessageFromClientCheckerFuncs(nil, handleMessageFromClientCheckerFuncs...)

	tx := makeTransaction(nodeRunners[0].Node().Keypair())
	c0.SendMessage(tx)

	select {
	case b := <-chanGotMessageFromClient:
		if b.MessageHash() != tx.GetHash() {
			t.Error("ballot does not match with transaction")
			return
		}
	case <-time.After(1 * time.Second):
		t.Error("failed to handle MessageFromClient")
		return
	}
}

// TestMemoryNetworkHandleMessageFromClientBroadcast checks, the message from
// client is broadcasted and the other validators can receive it's ballot
// correctly.
func TestMemoryNetworkHandleMessageFromClientBroadcast(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	nodeRunners := createNodeRunnersWithReady(4)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	c0 := nodeRunners[0].Network().GetClient(nodeRunners[0].Node().Endpoint())

	chanCancel := make(chan bool)
	chanGotBallot := make(chan Ballot)
	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			checker := c.(*NodeRunnerHandleBallotChecker)

			var ballot Ballot
			if ballot, err = NewBallotFromJSON(checker.Message.Data); err != nil {
				return
			}

			checker.Ballot = ballot
			chanGotBallot <- checker.Ballot
			return nil
		},
	}

	chanGotMessageFromClient := make(chan Ballot)
	var handleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageISAACReceiveMessage,
		CheckNodeRunnerHandleMessageSignBallot,
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			checker := c.(*NodeRunnerHandleMessageChecker)

			checker.NodeRunner.log.Debug("ballot from client will be broadcasted", "ballot", checker.Ballot.MessageHash())
			chanGotMessageFromClient <- checker.Ballot

			return nil
		},
		CheckNodeRunnerHandleMessageBroadcast,
	}
	nodeRunners[0].SetHandleMessageFromClientCheckerFuncs(nil, handleMessageFromClientCheckerFuncs...)

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(nil, handleBallotCheckerFuncs...)
	}

	tx := makeTransaction(nodeRunners[0].Node().Keypair())
	c0.SendMessage(tx)

	var sentBallot Ballot
	select {
	case sentBallot = <-chanGotMessageFromClient:
	}

	received := []Ballot{}

L:
	for {
		select {
		case <-chanCancel:
			return
		case b := <-chanGotBallot:
			received = append(received, b)
			if len(received) == 3 {
				break L
			}
		case <-time.After(3 * time.Second):
			t.Error("failed to receive ballots")
			break L
		}
	}

	for _, b := range received {
		if sentBallot.GetHash() != b.GetHash() {
			t.Errorf("got unknown ballot; '%s' != '%s'", sentBallot.GetHash(), b.GetHash())
			return
		}
		if sentBallot.MessageHash() != b.MessageHash() {
			t.Errorf("got unknown message; '%s' != '%s'", sentBallot.MessageHash(), b.MessageHash())
			return
		}
	}
}

// TestMemoryNetworkHandleBallotCheckIsNew checks, the already received message from
// client must be ignored.
func TestMemoryNetworkHandleMessageCheckHasMessage(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	nr := createNodeRunners(1)[0]
	go nr.Start()
	defer nr.Stop()

	c0 := nr.Network().GetClient(nr.Node().Endpoint())

	var wg sync.WaitGroup

	// we will send 3 tx
	wg.Add(3)

	var addedBallot []Ballot
	var foundErrors []error
	var handleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageISAACReceiveMessage,
		func(c sebakcommon.Checker, args ...interface{}) error {
			checker := c.(*NodeRunnerHandleMessageChecker)

			addedBallot = append(addedBallot, checker.Ballot)

			return nil
		},
	}

	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if n == 1 {
			defer wg.Done()
		}

		if err == nil {
			return
		}
		foundErrors = append(foundErrors, err)
	}

	nr.SetHandleMessageFromClientCheckerFuncs(deferFunc, handleMessageFromClientCheckerFuncs...)

	tx := makeTransaction(nr.Node().Keypair())
	c0.SendMessage(tx)
	c0.SendMessage(tx)
	c0.SendMessage(tx)

	wg.Wait()

	if len(addedBallot) != 1 {
		t.Error("only 1st tx must be added")
		return
	}

	// check error
	if len(foundErrors) != 2 {
		t.Error("2 `sebakerror.ErrorNewButKnownMessage` must be occurred")
		return
	}
	for _, err := range foundErrors {
		if err != sebakerror.ErrorNewButKnownMessage {
			t.Error("must raise error, `sebakerror.ErrorNewButKnownMessage`")
			return
		}
	}
}

// TestMemoryNetworkHandleMessageAddBallot checks, the each messages from
// client will be added.
func TestMemoryNetworkHandleMessageAddBallot(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	nodeRunners := createNodeRunners(2)
	nr0 := nodeRunners[0]
	nr1 := nodeRunners[1]
	go nr0.Start()
	defer nr0.Stop()

	c0 := nr0.Network().GetClient(nr0.Node().Endpoint())

	var wg sync.WaitGroup

	// we will send 3 tx
	wg.Add(3)

	var addedBallot []Ballot
	var foundErrors []error
	var handleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageISAACReceiveMessage,
		func(c sebakcommon.Checker, args ...interface{}) error {
			checker := c.(*NodeRunnerHandleMessageChecker)

			addedBallot = append(addedBallot, checker.Ballot)

			return nil
		},
	}

	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if n == 1 {
			defer wg.Done()
		}

		if err == nil {
			return
		}
		foundErrors = append(foundErrors, err)
	}

	nr0.SetHandleMessageFromClientCheckerFuncs(deferFunc, handleMessageFromClientCheckerFuncs...)

	c0.SendMessage(makeTransaction(nr0.Node().Keypair()))
	c0.SendMessage(makeTransaction(nr1.Node().Keypair()))
	c0.SendMessage(makeTransaction(nr1.Node().Keypair()))

	wg.Wait()

	if len(addedBallot) != 3 {
		t.Error("all tx must be added")
		return
	}

	// check error
	if len(foundErrors) != 0 {
		t.Error("error occurred", foundErrors)
		return
	}
}
