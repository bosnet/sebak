//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package runner

import (
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	ghandlers "github.com/gorilla/handlers"
	logging "github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/httpcache"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner/api"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/voting"
)

var DefaultHandleBaseBallotCheckerFuncs = []common.CheckerFunc{
	BallotUnmarshal,
	BallotNotFromKnownValidators,
	BallotCheckSYNC,
	BallotCheckBasis,
}

var DefaultHandleINITBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotValidateOperationBodyCollectTxFee,
	BallotValidateOperationBodyInflation,
	BallotGetMissingTransaction,
	INITBallotValidateTransactions,
	SIGNBallotBroadcast,
	TransitStateToSIGN,
}

var DefaultHandleSIGNBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	ExpiredInSIGN,
	ACCEPTBallotBroadcast,
	TransitStateToACCEPT,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	FinishedBallotStore,
}

var (
	corsAllowedOrigins ghandlers.CORSOption = ghandlers.AllowedOrigins([]string{"*"})
	corsAllowedMethods ghandlers.CORSOption = ghandlers.AllowedMethods([]string{"GET", "POST"})
	corsAllowedHeaders ghandlers.CORSOption = ghandlers.AllowedHeaders(
		[]string{"Content-Type", "X-Requested-With", "Cache-Control", "Access-Control"},
	)
)

type NodeRunner struct {
	sync.RWMutex

	localNode         *node.LocalNode
	policy            voting.ThresholdPolicy
	network           network.Network
	consensus         *consensus.ISAAC
	TransactionPool   *transaction.Pool
	connectionManager network.ConnectionManager
	storage           *storage.LevelDBBackend
	isaacStateManager *ISAACStateManager
	ballotSendRecord  *consensus.BallotSendRecord

	handleBaseBallotCheckerFuncs   []common.CheckerFunc
	handleINITBallotCheckerFuncs   []common.CheckerFunc
	handleSIGNBallotCheckerFuncs   []common.CheckerFunc
	handleACCEPTBallotCheckerFuncs []common.CheckerFunc

	handleBallotCheckerDeferFunc common.CheckerDeferFunc

	log logging.Logger

	InitialBalance common.Amount

	Conf                  common.Config
	nodeInfo              node.NodeInfo
	savingBlockOperations *SavingBlockOperations
	jsonrpcServer         *jsonrpcServer
}

func NewNodeRunner(
	localNode *node.LocalNode,
	policy voting.ThresholdPolicy,
	n network.Network,
	c *consensus.ISAAC,
	storage *storage.LevelDBBackend,
	tp *transaction.Pool,
	conf common.Config,
) (nr *NodeRunner, err error) {
	nr = &NodeRunner{
		localNode:       localNode,
		policy:          policy,
		network:         n,
		consensus:       c,
		TransactionPool: tp,
		storage:         storage,
		log:             log.New(logging.Ctx{"node": localNode.Alias()}),
		Conf:            conf,
	}
	nr.ballotSendRecord = consensus.NewBallotSendRecord(localNode.Alias())

	nr.localNode.SetBooting()

	nr.isaacStateManager = NewISAACStateManager(nr, conf)

	nr.policy.SetValidators(len(nr.localNode.GetValidators()))

	nr.connectionManager = c.ConnectionManager()
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)
	nr.savingBlockOperations = NewSavingBlockOperations(
		nr.Storage(),
		nr.Log(),
	)

	if err = nr.savingBlockOperations.Check(); err != nil {
		nr.log.Error("failed to check BlockOperations", "error", err)
		return
	}

	nr.SetHandleBaseBallotCheckerFuncs(DefaultHandleBaseBallotCheckerFuncs...)
	nr.SetHandleINITBallotCheckerFuncs(DefaultHandleINITBallotCheckerFuncs...)
	nr.SetHandleSIGNBallotCheckerFuncs(DefaultHandleSIGNBallotCheckerFuncs...)
	nr.SetHandleACCEPTBallotCheckerFuncs(DefaultHandleACCEPTBallotCheckerFuncs...)

	{
		// find common account
		var commonAccount *block.BlockAccount
		if commonAccount, err = GetCommonAccount(nr.storage); err != nil {
			return
		}
		nr.Conf.CommonAccountAddress = commonAccount.Address
		nr.log.Debug("common account found", "address", nr.Conf.CommonAccountAddress)

		// get the initial balance of geness account
		if nr.InitialBalance, err = GetGenesisBalance(nr.storage); err != nil {
			return
		}
		nr.log.Debug("initial balance found", "amount", nr.InitialBalance)
		nr.InitialBalance.Invariant()
	}

	nr.nodeInfo = NewNodeInfo(nr)
	if conf.JSONRPCEndpoint != nil {
		nr.jsonrpcServer = newJSONRPCServer(conf.JSONRPCEndpoint, nr.storage)
	}

	return
}

func (nr *NodeRunner) Ready() {
	rateLimitMiddlewareAPI := network.RateLimitMiddleware(nr.log, nr.Conf.RateLimitRuleAPI)
	if err := nr.network.AddMiddleware(network.RouterNameAPI, rateLimitMiddlewareAPI); err != nil {
		nr.log.Error("`network.RateLimitMiddleware` for `RouterNameAPI` has an error", "err", err)
		return
	}
	rateLimitMiddlewareNode := network.RateLimitMiddleware(nr.log, nr.Conf.RateLimitRuleNode)
	if err := nr.network.AddMiddleware(network.RouterNameNode, rateLimitMiddlewareNode); err != nil {
		nr.log.Error("`network.RateLimitMiddleware` for `RouterNameNode` has an error", "err", err)
		return
	}
	if err := nr.network.AddMiddleware(network.RouterNameMetric, rateLimitMiddlewareAPI); err != nil {
		nr.log.Error("`network.RateLimitMiddleware` for `RouterNameMetric` router has an error", "err", err)
		return
	}
	if err := nr.network.AddMiddleware(network.RouterNameDebug, rateLimitMiddlewareAPI); err != nil {
		nr.log.Error("`network.RateLimitMiddleware` for `RouterNameDebug` router has an error", "err", err)
		return
	}

	// BaseRouter's middlewares impact all sub routers.
	if err := nr.network.AddMiddleware("", network.RecoverMiddleware(nr.log)); err != nil {
		nr.log.Error("Middleware has an error", "err", err)
		return
	}

	cors := ghandlers.CORS(corsAllowedOrigins, corsAllowedMethods, corsAllowedHeaders)
	{ //CORS
		if err := nr.network.AddMiddleware(network.RouterNameAPI, cors); err != nil {
			nr.log.Error("failed to add middleware", "err", err)
			return
		}
		if err := nr.network.AddMiddleware("", cors); err != nil {
			nr.log.Error("failed to add middleware", "err", err)
			return
		}
	}

	// cache middleware
	var (
		cache     httpcache.Wrapper
		listCache httpcache.Wrapper
		baCache   httpcache.Wrapper
	)

	if nr.Conf.HTTPCacheAdapter == "" {
		// no use cache middleware
		cache = httpcache.NewNopClient()
		listCache = httpcache.NewNopClient()
		baCache = httpcache.NewNopClient()
		nr.log.Info("http cache is disabled")
	} else {
		cacheAdater, err := httpcache.NewAdapter(nr.Conf)
		if err != nil {
			nr.log.Error("failed to create new HTTP Cache adapter", "err", err)
			return
		}
		defaultCacheOptions := httpcache.WithOptions(
			httpcache.WithAdapter(cacheAdater),
			httpcache.WithStatusCode(404, 1*time.Second),
			httpcache.WithLogger(nr.log),
		)
		cache, err = httpcache.NewClient(defaultCacheOptions, httpcache.WithExpire(1*time.Minute))
		if err != nil {
			nr.log.Error("failed to create new middleware Cache", "err", err)
			return
		}
		listCache, err = httpcache.NewClient(defaultCacheOptions, httpcache.WithExpire(3*time.Second))
		if err != nil {
			nr.log.Error("failed to create new List cache", "err", err)
			return
		}
		baCache, err = httpcache.NewClient(defaultCacheOptions, httpcache.WithExpire(1*time.Second))
		if err != nil {
			nr.log.Error("failed to create new BlockAccount cache", "err", err)
			return
		}
		nr.log.Info("http cache is enabled")
	}

	// node handlers
	nodeHandler := NewNetworkHandlerNode(
		nr.localNode,
		nr.network,
		nr.storage,
		nr.consensus,
		nr.TransactionPool,
		network.UrlPathPrefixNode,
		nr.Conf,
	)

	nr.network.AddHandler(nodeHandler.HandlerURLPattern(NodeInfoHandlerPattern), nodeHandler.NodeInfoHandler)
	nr.network.AddHandler(nodeHandler.HandlerURLPattern(ConnectHandlerPattern), nodeHandler.ConnectHandler).
		Methods("POST").
		Headers("Content-Type", "application/json")
	nr.network.AddHandler(nodeHandler.HandlerURLPattern(MessageHandlerPattern), nodeHandler.MessageHandler).
		Methods("POST").
		Headers("Content-Type", "application/json")
	nr.network.AddHandler(nodeHandler.HandlerURLPattern(BallotHandlerPattern), nodeHandler.BallotHandler).
		Methods("POST").
		Headers("Content-Type", "application/json")
	nr.network.AddHandler(nodeHandler.HandlerURLPattern(GetBlocksPattern), nodeHandler.GetBlocksHandler).
		Methods("GET", "POST").
		MatcherFunc(common.PostAndJSONMatcher)
	nr.network.AddHandler(nodeHandler.HandlerURLPattern(GetTransactionPattern), nodeHandler.GetNodeTransactionsHandler).
		Methods("GET", "POST").
		MatcherFunc(common.PostAndJSONMatcher)

	nr.network.AddHandler(network.UrlPathPrefixMetric, promhttp.Handler().ServeHTTP)

	// api handlers
	apiHandler := api.NewNetworkHandlerAPI(
		nr.localNode,
		nr.network,
		nr.storage,
		network.UrlPathPrefixAPI,
		nr.nodeInfo,
	)
	apiHandler.GetLatestBlock = nr.Consensus().LatestBlock

	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountHandlerPattern),
		baCache.WrapHandlerFunc(apiHandler.GetAccountHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountsHandlerPattern),
		baCache.WrapHandlerFunc(apiHandler.GetAccountsHandler),
	).Methods("POST", "OPTIONS").MatcherFunc(common.PostAndJSONMatcher)
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountTransactionsHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetTransactionsByAccountHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionOperationHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetOperationsByTxHashOpIndexHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountOperationsHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetOperationsByAccountHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetFrozenAccountHandlerPattern),
		apiHandler.GetFrozenAccountsHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountFrozenAccountHandlerPattern),
		apiHandler.GetFrozenAccountsByAccountHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionByHashHandlerPattern),
		cache.WrapHandlerFunc(apiHandler.GetTransactionByHashHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionOperationsHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetOperationsByTxHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionStatusHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetTransactionStatusByHashHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.PostSubscribePattern),
		listCache.WrapHandlerFunc(apiHandler.PostSubscribeHandler),
	).Methods("POST", "OPTIONS")

	TransactionsHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {

			checkerFuncs := HandleTransactionCheckerFuncs

			if nr.Conf.WatcherMode == true {
				checkerFuncs = HandleTransactionCheckerForWatcherFuncs
			}

			apiHandler.PostTransactionsHandler(
				w, r,
				nodeHandler.ReceiveTransaction, checkerFuncs,
			)
			return
		}

		cache.WrapHandlerFunc(apiHandler.GetTransactionsHandler)(w, r)
		return
	}

	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionsHandlerPattern),
		TransactionsHandler,
	).Methods("GET", "POST", "OPTIONS").MatcherFunc(common.PostAndJSONMatcher)

	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetBlocksHandlerPattern),
		listCache.WrapHandlerFunc(apiHandler.GetBlocksHandler),
	).Methods("GET", "OPTIONS")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetBlockHandlerPattern),
		cache.WrapHandlerFunc(apiHandler.GetBlockHandler),
	).Methods("GET", "OPTIONS")

	// pprof
	if DebugPProf == true {
		nr.network.AddHandler(network.UrlPathPrefixDebug+"/pprof/cmdline", pprof.Cmdline)
		nr.network.AddHandler(network.UrlPathPrefixDebug+"/pprof/profile", pprof.Profile)
		nr.network.AddHandler(network.UrlPathPrefixDebug+"/pprof/symbol", pprof.Symbol)
		nr.network.AddHandler(network.UrlPathPrefixDebug+"/pprof/trace", pprof.Trace)
		nr.network.AddHandler(network.UrlPathPrefixDebug+"/pprof/*", pprof.Index)
	}

	nr.network.Ready()

	nr.network.AddHandler(api.GetNodeInfoPattern, apiHandler.GetNodeInfoHandler).Methods("GET", "OPTIONS")
}

func (nr *NodeRunner) Start() (err error) {
	nr.log.Debug("NodeRunner started")
	nr.Ready()

	go nr.handleMessages()
	go nr.ConnectValidators()
	go nr.InitRound()
	go nr.savingBlockOperations.Start()

	if nr.jsonrpcServer != nil {
		go func() {
			err = nr.jsonrpcServer.Start()
			if err != nil {
				log.Crit("failed to start jsonrpcServer", "error", err)
				nr.Stop()
			}
		}()
	}

	if err = nr.network.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunner) Stop() {
	nr.network.Stop()
	nr.isaacStateManager.Stop()
	if nr.jsonrpcServer != nil {
		nr.jsonrpcServer.Stop()
	}
}

func (nr *NodeRunner) Node() *node.LocalNode {
	return nr.localNode
}

func (nr *NodeRunner) NetworkID() []byte {
	return nr.Conf.NetworkID
}

func (nr *NodeRunner) Network() network.Network {
	return nr.network
}

func (nr *NodeRunner) Consensus() *consensus.ISAAC {
	return nr.consensus
}

func (nr *NodeRunner) ConnectionManager() network.ConnectionManager {
	return nr.connectionManager
}

func (nr *NodeRunner) Storage() *storage.LevelDBBackend {
	return nr.storage
}

func (nr *NodeRunner) Policy() voting.ThresholdPolicy {
	return nr.policy
}

func (nr *NodeRunner) Log() logging.Logger {
	return nr.log
}

func (nr *NodeRunner) SavingBlockOperations() *SavingBlockOperations {
	return nr.savingBlockOperations
}

func (nr *NodeRunner) BallotSendRecord() *consensus.BallotSendRecord {
	return nr.ballotSendRecord
}

func (nr *NodeRunner) ConnectValidators() {
	ticker := time.NewTicker(time.Millisecond * 5)
	for _ = range ticker.C {
		if !nr.network.IsReady() {
			continue
		}

		ticker.Stop()
		break
	}
	nr.log.Debug("current node is ready")
	nr.log.Debug("trying to connect to the validators", "validators", nr.localNode.GetValidators())
	nr.connectionManager.Start()
}

func (nr *NodeRunner) SetHandleBaseBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleBaseBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleINITBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleINITBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleSIGNBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleSIGNBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleACCEPTBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleACCEPTBallotCheckerFuncs = f
}

// Read from the network channel and forwards to `handleMessage`
func (nr *NodeRunner) handleMessages() {
	for message := range nr.network.ReceiveMessage() {
		nr.handleMessage(message)
	}
}

// Handles a single message received from a client
func (nr *NodeRunner) handleMessage(message common.NetworkMessage) {
	if message.IsEmpty() {
		nr.log.Error("got empty message")
		return
	}
	switch message.Type {
	case common.ConnectMessage:
		if _, err := node.NewValidatorFromString(message.Data); err != nil {
			nr.log.Error("invalid validator data was received", "error", err)
			return
		}
	case common.BallotMessage:
		nr.handleBallotMessage(message)
	default:
		nr.log.Error("got unknown message")
		return
	}
}

func (nr *NodeRunner) handleBallotMessage(message common.NetworkMessage) (err error) {
	nr.log.Debug("got ballot message")
	baseChecker := &BallotChecker{
		DefaultChecker:     common.DefaultChecker{Funcs: nr.handleBaseBallotCheckerFuncs},
		NodeRunner:         nr,
		LocalNode:          nr.localNode,
		Log:                nr.Log(),
		VotingHole:         voting.NOTYET,
		LatestBlockSources: []string{},
		Message:            message,
	}

	if err = common.RunChecker(baseChecker, nr.handleBallotCheckerDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Debug("failed to handle ballot", "error", err)
			return
		}
	}

	var checkerFuncs []common.CheckerFunc
	switch baseChecker.Ballot.State() {
	case ballot.StateINIT:
		checkerFuncs = DefaultHandleINITBallotCheckerFuncs
	case ballot.StateSIGN:
		checkerFuncs = DefaultHandleSIGNBallotCheckerFuncs
	case ballot.StateACCEPT:
		checkerFuncs = DefaultHandleACCEPTBallotCheckerFuncs
	}

	checker := &BallotChecker{
		DefaultChecker:     common.DefaultChecker{Funcs: checkerFuncs},
		NodeRunner:         nr,
		LocalNode:          nr.localNode,
		Ballot:             baseChecker.Ballot,
		VotingHole:         baseChecker.VotingHole,
		IsNew:              baseChecker.IsNew,
		IsMine:             baseChecker.IsMine,
		Log:                baseChecker.Log,
		LatestBlockSources: baseChecker.LatestBlockSources,
	}
	if err = common.RunChecker(checker, nr.handleBallotCheckerDeferFunc); err != nil {
		if stopped, ok := err.(common.CheckerStop); ok {
			nr.log.Debug(
				"stopped to handle ballot",
				"voting-basis", baseChecker.Ballot.VotingBasis(),
				"state", baseChecker.Ballot.State(),
				"reason", stopped.Error(),
			)
			err = nil
		} else {
			nr.log.Debug("failed to handle ballot", "error", err, "state", baseChecker.Ballot.State())
			return
		}
	}

	return
}

func (nr *NodeRunner) InitRound() {
	if nr.Conf.WatcherMode == true {
		return
	}
	// get latest blocks
	nr.consensus.SetLatestVotingBasis(voting.Basis{})

	nr.waitForConnectingEnoughNodes()
	nr.startStateManager()
}

func (nr *NodeRunner) waitForConnectingEnoughNodes() {
	ticker := time.NewTicker(time.Millisecond * 5)
	for _ = range ticker.C {
		if nr.connectionManager.IsReady() {
			ticker.Stop()
			break
		}
	}
	nr.log.Debug(
		"caught up network and connected to enough validators",
		"connected", nr.Policy().Connected(),
		"validators", nr.Policy().Validators(),
	)

	return
}

func (nr *NodeRunner) startStateManager() {
	nr.isaacStateManager.Start()
	nr.isaacStateManager.NextHeight()
	return
}

func (nr *NodeRunner) StopStateManager() {
	nr.isaacStateManager.Stop()
	return
}

func (nr *NodeRunner) TransitISAACState(basis voting.Basis, ballotState ballot.State) {
	nr.isaacStateManager.TransitISAACState(basis.Height, basis.Round, ballotState)
}

func (nr *NodeRunner) NextHeight() {
	nr.isaacStateManager.NextHeight()
}

func (nr *NodeRunner) RemoveSendRecordsLowerThanOrEqualHeight(height uint64) {
	nr.ballotSendRecord.RemoveLowerThanOrEqualHeight(height)
}

var NewBallotTransactionCheckerFuncs = []common.CheckerFunc{
	IsNew,
	BallotTransactionsSameSource,
}

func (nr *NodeRunner) proposeNewBallot(round uint64) (ballot.Ballot, error) {
	b := nr.consensus.LatestBlock()
	basis := voting.Basis{
		Round:     round,
		Height:    b.Height,
		BlockHash: b.Hash,
		TotalTxs:  b.TotalTxs,
		TotalOps:  b.TotalOps,
	}

	// collect incoming transactions from `Pool`
	availableTransactions := nr.TransactionPool.AvailableTransactions(nr.Conf.TxsLimit)
	nr.log.Debug("new round proposed", "block-basis", basis)

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker:        common.DefaultChecker{Funcs: NewBallotTransactionCheckerFuncs},
		NodeRunner:            nr,
		LocalNode:             nr.localNode,
		Transactions:          availableTransactions,
		CheckTransactionsOnly: true,
		VotingHole:            voting.NOTYET,
		transactionCache:      NewTransactionCache(nr.Storage(), nr.TransactionPool),
	}

	if err := common.RunChecker(transactionsChecker, common.DefaultDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Error("error occurred in BallotTransactionChecker", "error", err)
		}
	}

	// remove invalid transactions
	if len(transactionsChecker.InvalidTransactions()) > 0 {
		nr.TransactionPool.Remove(transactionsChecker.InvalidTransactions()...)
		nr.log.Debug(
			"invalid transactions removed from pool",
			"basis", basis,
			"invalid-transactions", len(transactionsChecker.InvalidTransactions()),
			"transactionpool", nr.TransactionPool.Len(),
		)
	}

	var validTransactions []transaction.Transaction
	var validTransactionHashes []string
	var ops int
	for _, hash := range transactionsChecker.ValidTransactions {
		var tx transaction.Transaction
		var found bool
		var err error
		if tx, found, err = transactionsChecker.transactionCache.Get(hash); err != nil {
			return ballot.Ballot{}, err
		} else if !found {
			return ballot.Ballot{}, errors.TransactionNotFound
		}

		if ops+len(tx.B.Operations) > nr.Conf.OpsInBallotLimit {
			continue
		}

		validTransactionHashes = append(validTransactionHashes, hash)
		validTransactions = append(validTransactions, tx)

		ops += len(tx.B.Operations)
		if ops == nr.Conf.OpsInBallotLimit {
			break
		}
	}

	proposerAddr := nr.consensus.SelectProposer(b.Height, round)
	blt := ballot.NewBallot(nr.localNode.Address(), proposerAddr, basis, validTransactionHashes)
	blt.SetVote(ballot.StateINIT, voting.YES)

	opc, err := ballot.NewCollectTxFeeFromBallot(*blt, nr.Conf.CommonAccountAddress, validTransactions...)
	if err != nil {
		return ballot.Ballot{}, err
	}

	opi, err := ballot.NewInflationFromBallot(*blt, nr.Conf.CommonAccountAddress, nr.InitialBalance)
	if err != nil {
		return ballot.Ballot{}, err
	}

	ptx, err := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)
	if err != nil {
		return ballot.Ballot{}, err
	}

	blt.SetProposerTransaction(ptx)
	blt.Sign(nr.localNode.Keypair(), nr.Conf.NetworkID)

	nr.log.Debug(
		"new ballot created",
		"ballot", blt.GetHash(),
		"basis", basis,
		"valid-transactions", len(validTransactions),
		"transactionpool", nr.TransactionPool.Len(),
	)

	nr.BroadcastBallot(*blt)

	return *blt, nil
}

func (nr *NodeRunner) NodeInfo() node.NodeInfo {
	return nr.nodeInfo
}

func (nr *NodeRunner) BroadcastBallot(b ballot.Ballot) {
	if nr.Node().State() == node.StateBOOTING && !nr.connectionManager.IsReady() {
		nr.waitForConnectingEnoughNodes()
	}

	state := consensus.ISAACState{
		Height:      b.VotingBasis().Height,
		Round:       b.VotingBasis().Round,
		BallotState: b.State(),
	}

	if nr.ballotSendRecord.Sent(state) {
		nr.Log().Debug(
			"return; already sent ballot in NodeRunner.BroadcastBallot",
			"ballot", b,
		)
		return
	}

	nr.Log().Debug(
		"broadcast ballot include itself",
		"ballot", b,
	)

	nr.ballotSendRecord.SetSent(state)

	go func() {
		encoded, _ := b.Serialize()
		nr.Network().MessageBroker().Receive(common.NewNetworkMessage(common.BallotMessage, encoded))
	}()

	nr.ConnectionManager().Broadcast(b)
}
