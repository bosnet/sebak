package runner

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	api "boscoin.io/sebak/lib/node/runner/node_api"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
)

type HelperAPIBallotTest struct {
	genesisBlock   block.Block
	initialBalance common.Amount
	commonAccount  *block.BlockAccount
	proposerNode   *node.LocalNode
	nr             *NodeRunner
}

func (p *HelperAPIBallotTest) Prepare() {
	nrs, _ := createTestNodeRunnersHTTP2Network(1)
	p.nr = nrs[0]
	p.proposerNode = p.nr.Node()
	p.nr.Consensus().SetProposerSelector(FixedSelector{p.proposerNode.Address()})

	// to stop broadacasting
	p.nr.Network().SetMessageBroker(&TestMessageBroker{network: p.nr.Network().(*network.HTTP2Network)})
	p.nr.isaacStateManager.Conf.BlockTime = time.Hour * 1
	p.nr.isaacStateManager.setBlockTimeBuffer()

	p.genesisBlock = block.GetGenesis(p.nr.Storage())
	p.commonAccount, _ = GetCommonAccount(p.nr.Storage())
	p.initialBalance, _ = GetGenesisBalance(p.nr.Storage())

	go p.nr.Start()

	ticker := time.NewTicker(time.Millisecond * 5)
	for _ = range ticker.C {
		if !p.nr.Network().IsReady() {
			continue
		}

		ticker.Stop()
		break
	}
}

func (p *HelperAPIBallotTest) Done() {
	p.nr.Stop()
}

func (p *HelperAPIBallotTest) MakeBallot(numberOfTxs int) (blt *ballot.Ballot) {
	txs := []transaction.Transaction{}
	txHashes := []string{}

	rd := voting.Basis{
		Round:     0,
		Height:    p.genesisBlock.Height,
		BlockHash: p.genesisBlock.Hash,
		TotalTxs:  p.genesisBlock.TotalTxs,
		TotalOps:  p.genesisBlock.TotalOps,
	}

	for i := 0; i < numberOfTxs; i++ {
		kpA := keypair.Random()
		accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve))
		accountA.MustSave(p.nr.Storage())

		kpB := keypair.Random()

		tx := transaction.MakeTransactionCreateAccount(networkID, kpA, kpB.Address(), common.Amount(1))
		tx.B.SequenceID = accountA.SequenceID
		tx.Sign(kpA, networkID)

		txHashes = append(txHashes, tx.GetHash())
		txs = append(txs, tx)

		// inject txs to `Pool`
		p.nr.TransactionPool.Add(tx)
	}

	blt = ballot.NewBallot(p.proposerNode.Address(), p.proposerNode.Address(), rd, txHashes)

	opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, p.commonAccount.Address, txs...)
	opi, _ := ballot.NewInflationFromBallot(*blt, p.commonAccount.Address, p.initialBalance)

	ptx, err := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	if err != nil {
		panic(err)
	}

	blt.SetProposerTransaction(ptx)
	blt.SetVote(ballot.StateINIT, voting.YES)
	blt.Sign(p.proposerNode.Keypair(), networkID)

	return
}

func (p *HelperAPIBallotTest) insertBallot(blt *ballot.Ballot) {
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
		Conf:           p.nr.Conf,
		LocalNode:      p.nr.Node(),
		Message:        ballotMessage,
		Log:            p.nr.Log(),
		VotingHole:     voting.NOTYET,
	}
	if err := common.RunChecker(baseChecker, common.DefaultDeferFunc); err != nil {
		panic(err)
	}

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleINITBallotCheckerFuncs},
		NodeRunner:     p.nr,
		Conf:           p.nr.Conf,
		LocalNode:      p.nr.Node(),
		Message:        ballotMessage,
		Ballot:         baseChecker.Ballot,
		VotingHole:     voting.NOTYET,
		Log:            p.nr.Log(),
	}
	if err := common.RunChecker(checker, common.DefaultDeferFunc); err != nil {
		panic(err)
	}
}

func TestAPIBallots(t *testing.T) {
	p := &HelperAPIBallotTest{}
	p.Prepare()
	defer p.Done()

	u, _ := url.Parse(p.nr.Node().Endpoint().String())
	u.Path = filepath.Join("/", network.RouterNameNode, GetBallotPattern)
	client := &http.Client{Transport: &http.Transport{}}

	{
		// request ballots; it should be empty
		req, err := http.NewRequest("GET", u.String(), nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		b, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, 0, len(b))
	}

	{
		blt := p.MakeBallot(0)
		p.insertBallot(blt)

		var collectedBallots []ballot.Ballot
		isaac := p.nr.Consensus()
		for _, rr := range isaac.RunningRounds {
			collectedBallots = append(collectedBallots, rr.Ballots...)
		}
		require.Equal(t, 1, len(collectedBallots))
		require.Equal(t, blt.GetHash(), collectedBallots[0].GetHash())

		{
			// request ballots
			req, err := http.NewRequest("GET", u.String(), nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
			require.NoError(t, err)
			require.Equal(t, 1, len(rbs[api.NodeItemBallot]))
			require.True(t, blt.Equal(rbs[api.NodeItemBallot][0].(ballot.Ballot)))
			require.Equal(t, blt.H.Signature, rbs[api.NodeItemBallot][0].(ballot.Ballot).H.Signature)
		}
	}

	{
		blt := p.MakeBallot(0)
		p.insertBallot(blt)

		var collectedBallots []ballot.Ballot
		isaac := p.nr.Consensus()
		for _, rr := range isaac.RunningRounds {
			collectedBallots = append(collectedBallots, rr.Ballots...)
		}
		require.Equal(t, 2, len(collectedBallots))
		require.Equal(t, blt.GetHash(), collectedBallots[1].GetHash())

		{
			// request ballots
			req, err := http.NewRequest("GET", u.String(), nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			rbs, err := unmarshalFromNodeItemResponseBody(resp.Body)
			require.NoError(t, err)
			require.Equal(t, 2, len(rbs[api.NodeItemBallot]))
			require.True(t, blt.Equal(rbs[api.NodeItemBallot][1].(ballot.Ballot)))
			require.Equal(t, blt.H.Signature, rbs[api.NodeItemBallot][1].(ballot.Ballot).H.Signature)
		}
	}
}
