package sebak

import (
	"fmt"
	"math/rand"
	"sync"

	"boscoin.io/sebak/lib/common"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := TestMakeTransaction(networkID, n)

	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(tx, a)
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
		Amount: sebakcommon.Amount(amount),
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

func TestGenerateNewCheckpoint() string {
	return uuid.New().String()
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
		fmt.Sprintf("%s-%s", TestGenerateNewCheckpoint(), TestGenerateNewCheckpoint()),
		ops...,
	)
	tx.Sign(srcKp, networkID)

	return
}

//
// Send a transaction to the network and wait indefinitely for consensus to be achieved
//
func doConsensus(nodeRunners []*NodeRunner, tx Transaction) []VotingStateStaging {
	var wg sync.WaitGroup
	wg.Add(len(nodeRunners))

	var messageDeferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}
		return
	}

	var dones []VotingStateStaging
	var finished []string
	var mutex = &sync.Mutex{}
	var ballotDeferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}
		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.LocalNode.Alias()); found {
			return
		}
		finished = append(finished, checker.LocalNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleMessageFromClientCheckerFuncs(messageDeferFunc)
		nr.SetHandleBallotCheckerFuncs(ballotDeferFunc)
	}

	nr0 := nodeRunners[0]
	client := nr0.Network().GetClient(nr0.Node().Endpoint())
	client.SendMessage(tx)
	wg.Wait()
	return dones
}
