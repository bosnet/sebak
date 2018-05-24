package sebak

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

func createNewMemoryServer() (*network.MemoryTransport, *util.Validator) {
	server := network.NewMemoryTransport()

	kp, _ := keypair.Random()
	validator, _ := util.NewValidator(kp.Address(), server.Endpoint(), "")
	validator.SetKeypair(kp)

	server.SetContext(context.WithValue(context.Background(), "currentNode", validator))

	return server, validator
}

func makeTransaction(kp *keypair.Full) (tx Transaction) {
	var ops []Operation
	ops = append(ops, MakeOperation(-1))

	txBody := TransactionBody{
		Source:     kp.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: util.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp)

	return
}

func createNodeRunners(n int) []*NodeRunner {
	var servers []*network.MemoryTransport
	var validators []*util.Validator
	for i := 0; i < n; i++ {
		s, v := createNewMemoryServer()
		servers = append(servers, s)
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
		p.SetValidators(uint64(len(v.GetValidators())) + 1)
		is, _ := NewISAAC(v, p)
		nr := NewNodeRunner(v, p, servers[i], is)
		nodeRunners = append(nodeRunners, nr)
	}

	return nodeRunners
}

// TestMemoryNetworkCreate checks, `NodeRunner` is correctly started and
// `GetNodeInfo` returns the validator information correctly.
func TestMemoryNetworkCreate(t *testing.T) {
	nodeRunners := createNodeRunners(4)
	for _, nr := range nodeRunners {
		go nr.Start()
		defer nr.Stop()
	}

	server := nodeRunners[0].TransportServer()
	for _, nr := range nodeRunners {
		client := server.GetClient(nr.Node().Endpoint())
		b, err := client.GetNodeInfo()
		if err != nil {
			t.Error(err)
			return
		}

		if rv, err := util.NewValidatorFromString(b); err != nil {
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
	nodeRunners := createNodeRunners(4)
	for _, nr := range nodeRunners {
		go nr.Start()
		defer nr.Stop()
	}

	c0 := nodeRunners[0].TransportServer().GetClient(nodeRunners[0].Node().Endpoint())

	chanGotMessageFromClient := make(chan Ballot)
	var handleMessageFromClientCheckerFuncs = []util.CheckerFunc{
		checkNodeRunnerHandleMessageTransactionUnmarshal,
		checkNodeRunnerHandleMessageISAACReceiveMessage,
		checkNodeRunnerHandleMessageSignBallot,
		func(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
			nr := target.(*NodeRunner)
			ballot := ctx.Value("ballot").(Ballot)

			nr.log.Debug("ballot from client will be broadcasted", "ballot", ballot.Message().GetHash())
			chanGotMessageFromClient <- ballot

			return ctx, nil
		},
	}

	nodeRunners[0].SetHandleMessageFromClientCheckerFuncs(nil, handleMessageFromClientCheckerFuncs...)

	tx := makeTransaction(nodeRunners[0].Node().Keypair())
	c0.SendMessage(tx)

	select {
	case b := <-chanGotMessageFromClient:
		if !b.Message().Equal(tx) {
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
	nodeRunners := createNodeRunners(4)
	for _, nr := range nodeRunners {
		go nr.Start()
		defer nr.Stop()
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
				if nr.ConnectionManager().CountConnected() != 3 {
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

	c0 := nodeRunners[0].TransportServer().GetClient(nodeRunners[0].Node().Endpoint())

	chanCancel := make(chan bool)
	chanGotBallot := make(chan Ballot)
	var handleBallotCheckerFuncs = []util.CheckerFunc{
		func(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
			var err error
			message, ok := args[0].(network.Message)
			if !ok {
				err = errors.New("invalid `network.Message`")
				t.Error(err)
				chanCancel <- true

				return ctx, err
			}

			var ballot Ballot
			if ballot, err = NewBallotFromJSON(message.Data); err != nil {
				return ctx, err
			}

			chanGotBallot <- ballot
			return ctx, nil
		},
	}

	chanGotMessageFromClient := make(chan Ballot)
	var handleMessageFromClientCheckerFuncs = []util.CheckerFunc{
		checkNodeRunnerHandleMessageTransactionUnmarshal,
		checkNodeRunnerHandleMessageISAACReceiveMessage,
		checkNodeRunnerHandleMessageSignBallot,
		func(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
			ballot := ctx.Value("ballot").(Ballot)

			nr := target.(*NodeRunner)
			nr.log.Debug("ballot from client will be broadcasted", "ballot", ballot.Message().GetHash())
			chanGotMessageFromClient <- ballot

			return ctx, nil
		},
		checkNodeRunnerHandleMessageBroadcast,
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
		if !sentBallot.Message().Equal(b.Message()) {
			t.Errorf("got unknown message; '%s' != '%s'", sentBallot.Message(), b.Message())
			return
		}
	}
}

// TestMemoryNetworkHandleBallotCheckIsNew checks, the already received message from
// client must be ignored.
func TestMemoryNetworkHandleMessageCheckHasMessage(t *testing.T) {
	nr := createNodeRunners(1)[0]
	go nr.Start()
	defer nr.Stop()

	c0 := nr.TransportServer().GetClient(nr.Node().Endpoint())

	var wg sync.WaitGroup

	// we will send 3 tx
	wg.Add(3)

	var addedBallot []Ballot
	var foundErrors []error
	var handleMessageFromClientCheckerFuncs = []util.CheckerFunc{
		checkNodeRunnerHandleMessageTransactionUnmarshal,
		checkNodeRunnerHandleMessageISAACReceiveMessage,
		func(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
			ballot := ctx.Value("ballot").(Ballot)

			addedBallot = append(addedBallot, ballot)

			return ctx, nil
		},
	}

	var deferFunc util.DeferFunc = func(f util.CheckerFunc, ctx context.Context, err error) {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(checkNodeRunnerHandleMessageISAACReceiveMessage).Pointer() {
			defer wg.Done()
		}

		if err == nil {
			return
		}
		foundErrors = append(foundErrors, err)
	}
	ctx := context.WithValue(context.Background(), "deferFunc", deferFunc)

	nr.SetHandleMessageFromClientCheckerFuncs(ctx, handleMessageFromClientCheckerFuncs...)

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
	nodeRunners := createNodeRunners(2)
	nr0 := nodeRunners[0]
	nr1 := nodeRunners[1]
	go nr0.Start()
	defer nr0.Stop()

	c0 := nr0.TransportServer().GetClient(nr0.Node().Endpoint())

	var wg sync.WaitGroup

	// we will send 3 tx
	wg.Add(3)

	var addedBallot []Ballot
	var foundErrors []error
	var handleMessageFromClientCheckerFuncs = []util.CheckerFunc{
		checkNodeRunnerHandleMessageTransactionUnmarshal,
		checkNodeRunnerHandleMessageISAACReceiveMessage,
		func(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
			ballot := ctx.Value("ballot").(Ballot)

			addedBallot = append(addedBallot, ballot)

			return ctx, nil
		},
	}

	var deferFunc util.DeferFunc = func(f util.CheckerFunc, ctx context.Context, err error) {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(checkNodeRunnerHandleMessageISAACReceiveMessage).Pointer() {
			defer wg.Done()
		}

		if err == nil {
			return
		}
		foundErrors = append(foundErrors, err)
	}
	ctx := context.WithValue(context.Background(), "deferFunc", deferFunc)

	nr0.SetHandleMessageFromClientCheckerFuncs(ctx, handleMessageFromClientCheckerFuncs...)

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
