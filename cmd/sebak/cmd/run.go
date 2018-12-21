package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"github.com/ulule/limiter"
	"golang.org/x/net/http2"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/metrics"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/sync"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/version"
)

const (
	defaultBindURL   string      = "https://0.0.0.0:12345"
	defaultLogFormat string      = "terminal"
	defaultLogLevel  logging.Lvl = logging.LvlInfo
)

var (
	flagBindURL                    string = common.GetENVValue("SEBAK_BIND", defaultBindURL)
	flagBlockTime                  string = common.GetENVValue("SEBAK_BLOCK_TIME", "5s")
	flagBlockTimeDelta             string = common.GetENVValue("SEBAK_BLOCK_TIME_DELTA", "1s")
	flagDebugPProf                 bool   = common.GetENVValue("SEBAK_DEBUG_PPROF", "0") == "1"
	flagKPSecretSeed               string = common.GetENVValue("SEBAK_SECRET_SEED", "")
	flagLog                        string = common.GetENVValue("SEBAK_LOG", "")
	flagLogRotateMaxSize           string = common.GetENVValue("SEBAK_LOG_ROTATE_MAX_SIZE", "100")
	flagLogRotateMaxCount          string = common.GetENVValue("SEBAK_LOG_ROTATE_MAX_COUNT", "0")
	flagLogRotateMaxDays           string = common.GetENVValue("SEBAK_LOG_ROTATE_MAX_DAYS", "0")
	flagLogRotateUncompress        bool   = common.GetENVValue("SEBAK_LOG_ROTATE_UNCOMPRESS", "0") == "1"
	flagHTTPLog                    string = common.GetENVValue("SEBAK_HTTP_LOG", "")
	flagHTTPLogRotateMaxSize       string = common.GetENVValue("SEBAK_HTTP_LOG_ROTATE_MAX_SIZE", "100")
	flagHTTPLogRotateMaxCount      string = common.GetENVValue("SEBAK_HTTP_LOG_ROTATE_MAX_COUNT", "0")
	flagHTTPLogRotateMaxDays       string = common.GetENVValue("SEBAK_HTTP_LOG_ROTATE_MAX_DAYS", "0")
	flagHTTPLogRotateUncompress    bool   = common.GetENVValue("SEBAK_HTTP_LOG_ROTATE_UNCOMPRESS", "0") == "1"
	flagLogLevel                   string = common.GetENVValue("SEBAK_LOG_LEVEL", defaultLogLevel.String())
	flagLogFormat                  string = common.GetENVValue("SEBAK_LOG_FORMAT", defaultLogFormat)
	flagNetworkID                  string = common.GetENVValue("SEBAK_NETWORK_ID", "")
	flagPublishURL                 string = common.GetENVValue("SEBAK_PUBLISH", "")
	flagSyncCheckInterval          string = common.GetENVValue("SEBAK_SYNC_CHECK_INTERVAL", "30s")
	flagSyncFetchTimeout           string = common.GetENVValue("SEBAK_SYNC_FETCH_TIMEOUT", "1m")
	flagSyncPoolSize               string = common.GetENVValue("SEBAK_SYNC_POOL_SIZE", "300")
	flagSyncRetryInterval          string = common.GetENVValue("SEBAK_SYNC_RETRY_INTERVAL", "10s")
	flagSyncCheckPrevBlockInterval string = common.GetENVValue("SEBAK_SYNC_CHECK_PREVBLOCK", "30s")
	flagThreshold                  string = common.GetENVValue("SEBAK_THRESHOLD", "67")
	flagTimeoutACCEPT              string = common.GetENVValue("SEBAK_TIMEOUT_ACCEPT", "2s")
	flagTimeoutALLCONFIRM          string = common.GetENVValue("SEBAK_TIMEOUT_ALLCONFIRM", "30s")
	flagTimeoutINIT                string = common.GetENVValue("SEBAK_TIMEOUT_INIT", "2s")
	flagTimeoutSIGN                string = common.GetENVValue("SEBAK_TIMEOUT_SIGN", "2s")
	flagTLSCertFile                string = common.GetENVValue("SEBAK_TLS_CERT", "sebak.crt")
	flagTLSKeyFile                 string = common.GetENVValue("SEBAK_TLS_KEY", "sebak.key")
	flagUnfreezingPeriod           string = common.GetENVValue("SEBAK_UNFREEZING_PERIOD", strconv.FormatUint(common.UnfreezingPeriod, 10))
	flagValidators                 string = common.GetENVValue("SEBAK_VALIDATORS", "")
	flagVerbose                    bool   = common.GetENVValue("SEBAK_VERBOSE", "0") == "1"
	flagCongressAddress            string = common.GetENVValue("SEBAK_CONGRESS_ADDR", "")
	flagJSONRPCBindURL             string = common.GetENVValue("SEBAK_JSONRPC_BIND", common.DefaultJSONRPCBindURL)

	flagRateLimitAPI        cmdcommon.ListFlags // "SEBAK_RATE_LIMIT_API"
	flagRateLimitNode       cmdcommon.ListFlags // "SEBAK_RATE_LIMIT_NODE"
	flagStorageConfigString string

	flagHTTPCacheAdapter    string = common.GetENVValue("SEBAK_HTTP_CACHE_ADAPTER", "")
	flagHTTPCachePoolSize   string = common.GetENVValue("SEBAK_HTTP_CACHE_POOL_SIZE", "10000")
	flagHTTPCacheRedisAddrs string = common.GetENVValue("SEBAK_HTTP_CACHE_REDIS_ADDRS", "")

	flagOperationsLimit         string = common.GetENVValue("SEBAK_OPERATIONS_LIMIT", strconv.Itoa(common.DefaultOperationsInTransactionLimit))
	flagTransactionsLimit       string = common.GetENVValue("SEBAK_TRANSACTIONS_LIMIT", strconv.Itoa(common.DefaultTransactionsInBallotLimit))
	flagOperationsInBallotLimit string = common.GetENVValue("SEBAK_OPERATIONS_IN_BALLOT_LIMIT", strconv.Itoa(common.DefaultOperationsInBallotLimit))
	flagTxPoolLimit             string = common.GetENVValue("SEBAK_TX_POOL_LIMIT", strconv.Itoa(common.DefaultTxPoolLimit))

	flagWatcherMode   bool   = common.GetENVValue("SEBAK_WATCHER_MODE", "0") == "1"
	flagWatchInterval string = common.GetENVValue("SEBAK_WATCH_INTERVAL", "5s")

	flagDiscovery cmdcommon.ListFlags // "SEBAK_DISCOVERY"
)

var (
	nodeCmd                 *cobra.Command
	bindEndpoint            *common.Endpoint
	blockTime               time.Duration
	blockTimeDelta          time.Duration
	kp                      *keypair.Full
	localNode               *node.LocalNode
	publishEndpoint         *common.Endpoint
	rateLimitRuleAPI        common.RateLimitRule
	rateLimitRuleNode       common.RateLimitRule
	storageConfig           *storage.Config
	syncCheckInterval       time.Duration
	syncFetchTimeout        time.Duration
	syncPoolSize            uint64
	syncRetryInterval       time.Duration
	threshold               int
	timeoutACCEPT           time.Duration
	timeoutALLCONFIRM       time.Duration
	timeoutINIT             time.Duration
	timeoutSIGN             time.Duration
	validators              []*node.Validator
	httpCacheAdapter        string
	httpCachePoolSize       int
	httpCacheRedisAddrs     map[string]string
	operationsLimit         uint64
	transactionsLimit       uint64
	operationsInBallotLimit uint64
	txPoolClientLimit       uint64
	txPoolNodeLimit         uint64
	syncCheckPrevBlock      time.Duration
	jsonrpcbindEndpoint     *common.Endpoint
	watchInterval           time.Duration
	discoveryEndpoints      []*common.Endpoint

	logLevel              logging.Lvl
	log                   logging.Logger = logging.New("module", "main")
	logRotateMaxSize      int
	logRotateMaxCount     int
	logRotateMaxDays      int
	httpLogRotateMaxSize  int
	httpLogRotateMaxCount int
	httpLogRotateMaxDays  int
)

func init() {
	var err error
	var flagGenesis string

	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run sebak node",
		Run: func(c *cobra.Command, args []string) {
			// If `--genesis` was provided, perfom `sebak genesis` before starting the node
			// This allows one-step startup from scratch, quite useful for testing
			if len(flagGenesis) > 0 {
				genesisKP, commonKP, balance, err := parseGenesisOptionFromCSV(flagGenesis)
				if err != nil {
					cmdcommon.PrintFlagsError(nodeCmd, "--genesis", err)
				}

				flagName, err := makeGenesisBlock(
					genesisKP,
					commonKP,
					flagNetworkID,
					balance,
					flagStorageConfigString,
					log,
				)
				if len(flagName) != 0 || err != nil {
					cmdcommon.PrintFlagsError(c, flagName, err)
				}
			}

			parseFlagsNode()

			if err = runNode(); err != nil {
				// TODO: Handle errors here
				// We should handle the error correctly, however since we don't currently
				// shut down the HTTP server correctly, we get an error,
				// and we need the binary to exit with a successfull error code for
				// code coverage in integration test to work.
				log.Error("Node exited with error", "error", err)
			}
		},
	}

	// storage
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--storage", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--storage", err)
	}
	flagStorageConfigString = common.GetENVValue("SEBAK_STORAGE", fmt.Sprintf("file://%s/db", currentDirectory))

	nodeCmd.Flags().StringVar(&flagGenesis, "genesis", flagGenesis, "performs the 'genesis' command before running node. Syntax: key[,balance]")
	nodeCmd.Flags().StringVar(&flagKPSecretSeed, "secret-seed", flagKPSecretSeed, "secret seed of this node")
	nodeCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	nodeCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	nodeCmd.Flags().StringVar(&flagLogFormat, "log-format", flagLogFormat, "log format, {terminal, json}")
	nodeCmd.Flags().StringVar(&flagLog, "log", flagLog, "set log file")
	nodeCmd.Flags().StringVar(&flagLogRotateMaxSize, "log-rotate-max-size", flagLogRotateMaxSize, "max size of rotate log")
	nodeCmd.Flags().StringVar(&flagLogRotateMaxCount, "log-rotate-max-count", flagLogRotateMaxCount, "max count of rotated logs")
	nodeCmd.Flags().StringVar(&flagLogRotateMaxDays, "log-rotate-max-days", flagLogRotateMaxDays, "max days of rotated logs")
	nodeCmd.Flags().BoolVar(&flagLogRotateUncompress, "log-rotate-uncompress", flagLogRotateUncompress, "disable compression of rotate log")
	nodeCmd.Flags().StringVar(&flagHTTPLog, "http-log", flagHTTPLog, "set log file for HTTP request")
	nodeCmd.Flags().StringVar(&flagHTTPLogRotateMaxSize, "http-log-rotate-max-size", flagHTTPLogRotateMaxSize, "max size of rotate http log")
	nodeCmd.Flags().StringVar(&flagHTTPLogRotateMaxCount, "http-log-rotate-max-count", flagHTTPLogRotateMaxCount, "max count of rotated http logs")
	nodeCmd.Flags().StringVar(&flagHTTPLogRotateMaxDays, "http-log-rotate-max-days", flagHTTPLogRotateMaxDays, "max days of rotated http logs")
	nodeCmd.Flags().BoolVar(&flagHTTPLogRotateUncompress, "http-log-rotate-uncompress", flagHTTPLogRotateUncompress, "disable compression of rotate http log")
	nodeCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	nodeCmd.Flags().StringVar(&flagBindURL, "bind", flagBindURL, "bind to listen on")
	nodeCmd.Flags().StringVar(&flagJSONRPCBindURL, "jsonrpc-bind", flagJSONRPCBindURL, "bind to listen on for jsonrpc")
	nodeCmd.Flags().StringVar(&flagPublishURL, "publish", flagPublishURL, "endpoint url for other nodes")
	nodeCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	nodeCmd.Flags().StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	nodeCmd.Flags().StringVar(&flagTLSKeyFile, "tls-key", flagTLSKeyFile, "tls key file")
	nodeCmd.Flags().StringVar(&flagValidators, "validators", flagValidators, "set validator: <endpoint url>?address=<public address>[&alias=<alias>] [ <validator>...]")
	nodeCmd.Flags().StringVar(&flagThreshold, "threshold", flagThreshold, "threshold")
	nodeCmd.Flags().StringVar(&flagTimeoutINIT, "timeout-init", flagTimeoutINIT, "timeout of the init state")
	nodeCmd.Flags().StringVar(&flagTimeoutSIGN, "timeout-sign", flagTimeoutSIGN, "timeout of the sign state")
	nodeCmd.Flags().StringVar(&flagTimeoutACCEPT, "timeout-accept", flagTimeoutACCEPT, "timeout of the accept state")
	nodeCmd.Flags().StringVar(&flagTimeoutALLCONFIRM, "timeout-allconfirm", flagTimeoutALLCONFIRM, "timeout of the allconfirm state")
	nodeCmd.Flags().StringVar(&flagBlockTime, "block-time", flagBlockTime, "block creation time")
	nodeCmd.Flags().StringVar(&flagBlockTimeDelta, "block-time-delta", flagBlockTimeDelta, "variation period of block time")
	nodeCmd.Flags().StringVar(&flagUnfreezingPeriod, "unfreezing-period", flagUnfreezingPeriod, "how long freezing must last")
	nodeCmd.Flags().StringVar(&flagOperationsLimit, "operations-limit", flagOperationsLimit, "operations limit in a transaction")
	nodeCmd.Flags().StringVar(&flagTransactionsLimit, "transactions-limit", flagTransactionsLimit, "transactions limit in a ballot")
	nodeCmd.Flags().StringVar(&flagOperationsInBallotLimit, "operations-in-ballot-limit", flagOperationsInBallotLimit, "operations limit in a ballot")
	nodeCmd.Flags().StringVar(&flagTxPoolLimit, "txpool-limit", flagTxPoolLimit, "transaction pool limit: <client-side>[,<node-side>] (0= no limit)")
	nodeCmd.Flags().Var(
		&flagRateLimitAPI,
		"rate-limit-api",
		fmt.Sprintf("rate limit for %s: [<ip>=]<limit>-<period>, ex) '10-S' '3.3.3.3=1000-M'", network.UrlPathPrefixAPI),
	)
	nodeCmd.Flags().Var(
		&flagRateLimitNode,
		"rate-limit-node",
		fmt.Sprintf("rate limit for %s: [<ip>=]<limit>-<period>, ex) '10-S' '3.3.3.3=1000-M'", network.UrlPathPrefixNode),
	)

	nodeCmd.Flags().BoolVar(&flagDebugPProf, "debug-pprof", flagDebugPProf, "set debug pprof")

	nodeCmd.Flags().StringVar(&flagSyncPoolSize, "sync-pool-size", flagSyncPoolSize, "sync pool size")
	nodeCmd.Flags().StringVar(&flagSyncFetchTimeout, "sync-fetch-timeout", flagSyncFetchTimeout, "sync fetch timeout")
	nodeCmd.Flags().StringVar(&flagSyncRetryInterval, "sync-retry-interval", flagSyncRetryInterval, "sync retry interval")
	nodeCmd.Flags().StringVar(&flagSyncCheckInterval, "sync-check-interval", flagSyncCheckInterval, "sync check interval")
	nodeCmd.Flags().StringVar(&flagSyncCheckPrevBlockInterval, "sync-check-prevblock", flagSyncCheckPrevBlockInterval, "sync check interval for previous block")

	nodeCmd.Flags().StringVar(&flagHTTPCacheAdapter, "http-cache-adapter", flagHTTPCacheAdapter, "http cache adapter: ex) 'mem'")
	nodeCmd.Flags().StringVar(&flagHTTPCachePoolSize, "http-cache-pool-size", flagHTTPCachePoolSize, "http cache pool size")
	nodeCmd.Flags().StringVar(&flagHTTPCacheRedisAddrs, "http-cache-redis-addrs", flagHTTPCacheRedisAddrs, "http cache redis address")

	nodeCmd.Flags().StringVar(&flagCongressAddress, "set-congress-address", flagCongressAddress, "set congress address")
	nodeCmd.Flags().BoolVar(&flagWatcherMode, "watcher-mode", flagWatcherMode, "watcher mode")
	nodeCmd.Flags().StringVar(&flagWatchInterval, "watch-interval", flagWatchInterval, "watch interval")
	nodeCmd.Flags().Var(&flagDiscovery, "discovery", "initial endpoint for discovery")

	rootCmd.AddCommand(nodeCmd)
}

func parseFlagRateLimit(l cmdcommon.ListFlags, defaultRate limiter.Rate) (rule common.RateLimitRule, err error) {
	if len(l) < 1 {
		rule = common.NewRateLimitRule(defaultRate)
		return
	}

	var givenRate limiter.Rate

	byIPAddress := map[string]limiter.Rate{}
	for _, s := range l {
		sl := strings.SplitN(s, "=", 2)

		var ip, r string
		if len(sl) < 2 {
			r = s
		} else {
			ip = sl[0]
			r = sl[1]
		}

		if len(ip) > 0 {
			if _, _, err = net.ParseCIDR(ip); err != nil {
				if net.ParseIP(ip) == nil {
					err = fmt.Errorf("invalid ip or cirdr address")
					return
				}
			}
		}

		var rate limiter.Rate
		if rate, err = limiter.NewRateFromFormatted(r); err != nil {
			return
		}

		if len(ip) > 0 {
			byIPAddress[ip] = rate
		} else {
			givenRate = rate
		}
	}

	if givenRate.Period < 1 && givenRate.Limit < 1 {
		givenRate = defaultRate
	}

	rule = common.NewRateLimitRule(givenRate)
	rule.ByIPAddress = byIPAddress

	return
}

func parseFlagValidators(s string) (vs []*node.Validator, err error) {
	splitted := strings.Fields(strings.TrimSpace(s))
	if len(splitted) < 1 {
		err = fmt.Errorf("must be given")
		return
	}

	for _, v := range splitted {
		if v == "self" {
			continue
		}

		if _, err = keypair.Parse(v); err != nil {
			return
		}

		var validator *node.Validator
		if validator, err = node.NewValidator(v, nil, ""); err != nil {
			return
		}
		vs = append(vs, validator)
	}

	return
}

func parseFlagDiscovery(l cmdcommon.ListFlags) (endpoints []*common.Endpoint, err error) {
	if len(l) < 1 {
		return
	}

	var endpoint *common.Endpoint
	for _, s := range l {
		if endpoint, err = common.NewEndpointFromString(s); err != nil {
			return
		}

		var found bool
		for _, e := range endpoints {
			if endpoint.Equal(e) {
				found = true
				break
			}
		}
		if found {
			continue
		}

		endpoints = append(endpoints, endpoint)
	}

	return
}

func parseLogRotateMaxSize(option, value string) int {
	if len(value) < 1 {
		return 0
	}

	var s int64
	var err error
	if s, err = strconv.ParseInt(value, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, option, err)
	} else if s < 0 {
		cmdcommon.PrintFlagsError(nodeCmd, option, fmt.Errorf("greater than 0"))
	}

	return int(s)
}

func parseLogRotateMaxCount(option, value string) int {
	if len(value) < 1 {
		return 100 // 100M
	}

	var s int64
	var err error
	if s, err = strconv.ParseInt(value, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, option, err)
	} else if s < 0 {
		cmdcommon.PrintFlagsError(nodeCmd, option, fmt.Errorf("greater than 0"))
	}

	return int(s)
}

func parseLogRotateMaxDays(option, value string) int {
	if len(value) < 1 {
		return 0
	}

	var s int64
	var err error
	if s, err = strconv.ParseInt(value, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, option, err)
	} else if s < 0 {
		cmdcommon.PrintFlagsError(nodeCmd, option, fmt.Errorf("greater than 0"))
	}

	return int(s)
}

func parseFlagsNode() {
	var err error

	if len(flagNetworkID) < 1 {
		cmdcommon.PrintFlagsError(nodeCmd, "--network-id", errors.New("--network-id must be given"))
	}
	if len(flagKPSecretSeed) < 1 {
		cmdcommon.PrintFlagsError(nodeCmd, "--secret-seed", errors.New("must be given"))
	}

	var parsedKP keypair.KP
	parsedKP, err = keypair.Parse(flagKPSecretSeed)
	if err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--secret-seed", err)
	} else {
		kp = parsedKP.(*keypair.Full)
	}

	if p, err := common.ParseEndpoint(flagBindURL); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--bind", err)
	} else {
		bindEndpoint = p
		flagBindURL = bindEndpoint.String()
	}

	if strings.ToLower(bindEndpoint.Scheme) == "https" {
		if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
			cmdcommon.PrintFlagsError(nodeCmd, "--tls-cert", err)
		}
		if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
			cmdcommon.PrintFlagsError(nodeCmd, "--tls-key", err)
		}
	}

	queries := bindEndpoint.Query()
	queries.Add("TLSCertFile", flagTLSCertFile)
	queries.Add("TLSKeyFile", flagTLSKeyFile)
	queries.Add("IdleTimeout", "3s")
	bindEndpoint.RawQuery = queries.Encode()

	if len(flagPublishURL) > 0 {
		if p, err := common.ParseEndpoint(flagPublishURL); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--publish", err)
		} else {
			publishEndpoint = p
			flagPublishURL = publishEndpoint.String()
		}
	} else {
		publishEndpoint = &common.Endpoint{}
		*publishEndpoint = *bindEndpoint
		publishEndpoint.Host = fmt.Sprintf("localhost:%s", publishEndpoint.Port())
		flagPublishURL = publishEndpoint.String()
	}

	if len(flagJSONRPCBindURL) > 0 { // jsonrpc
		if p, err := common.ParseEndpoint(flagJSONRPCBindURL); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--jsonrpc-bind", err)
		} else {
			jsonrpcbindEndpoint = p
			flagJSONRPCBindURL = jsonrpcbindEndpoint.String()
		}

		if strings.ToLower(jsonrpcbindEndpoint.Scheme) == "https" {
			if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
				cmdcommon.PrintFlagsError(nodeCmd, "--tls-cert", err)
			}
			if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
				cmdcommon.PrintFlagsError(nodeCmd, "--tls-key", err)
			}
		}

		queries := jsonrpcbindEndpoint.Query()
		queries.Add("TLSCertFile", flagTLSCertFile)
		queries.Add("TLSKeyFile", flagTLSKeyFile)
		queries.Add("IdleTimeout", "3s")
		jsonrpcbindEndpoint.RawQuery = queries.Encode()
	}

	if validators, err = parseFlagValidators(flagValidators); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--validators", err)
	}

	if storageConfig, err = storage.NewConfigFromString(flagStorageConfigString); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--storage", err)
	}

	timeoutINIT = getTimeDuration(flagTimeoutINIT, common.DefaultTimeoutINIT, "--timeout-init")
	timeoutSIGN = getTimeDuration(flagTimeoutSIGN, common.DefaultTimeoutSIGN, "--timeout-sign")
	timeoutACCEPT = getTimeDuration(flagTimeoutACCEPT, common.DefaultTimeoutACCEPT, "--timeout-accept")
	timeoutALLCONFIRM = getTimeDuration(flagTimeoutALLCONFIRM, common.DefaultTimeoutALLCONFIRM, "--timeout-allconfirm")
	blockTime = getTimeDuration(flagBlockTime, common.DefaultBlockTime, "--block-time")
	blockTimeDelta = getTimeDuration(flagBlockTimeDelta, common.DefaultBlockTimeDelta, "--block-time-delta")

	if transactionsLimit, err = strconv.ParseUint(flagTransactionsLimit, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--transactions-limit", err)
	}

	if operationsLimit, err = strconv.ParseUint(flagOperationsLimit, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--operations-limit", err)
	}

	if operationsInBallotLimit, err = strconv.ParseUint(flagOperationsInBallotLimit, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--operations-in-ballot-limit", err)
	}

	var tmpThreshold uint64
	if tmpThreshold, err = strconv.ParseUint(flagThreshold, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--threshold", err)
	} else {
		threshold = int(tmpThreshold)
	}

	// tx pool limits (client,node)
	{
		limits := strings.Split(flagTxPoolLimit, ",")
		if len(limits) > 2 {
			cmdcommon.PrintFlagsError(nodeCmd, "--txpool-limit", fmt.Errorf("wrong format: format:<client-limit>[,<node-limit]"))
		}
	L:
		for i, l := range limits {
			switch i {
			case 0:
				if txPoolClientLimit, err = strconv.ParseUint(l, 10, 64); err != nil {
					cmdcommon.PrintFlagsError(nodeCmd, "--txpool-limit", err)
				}
			case 1:
				if txPoolNodeLimit, err = strconv.ParseUint(l, 10, 64); err != nil {
					cmdcommon.PrintFlagsError(nodeCmd, "--txpool-limit", err)
				}
			default:
				break L
			}
		}
	}

	if common.UnfreezingPeriod, err = strconv.ParseUint(flagUnfreezingPeriod, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--unfreezing-period", err)
	}

	if syncPoolSize, err = strconv.ParseUint(flagSyncPoolSize, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--sync-pool-size", err)
	}

	syncRetryInterval = getTimeDuration(flagSyncRetryInterval, sync.RetryInterval, "--sync-retry-interval")
	syncFetchTimeout = getTimeDuration(flagSyncFetchTimeout, sync.FetchTimeout, "--sync-fetch-timeout")
	syncCheckInterval = getTimeDuration(flagSyncCheckInterval, sync.CheckBlockHeightInterval, "--sync-check-interval")
	syncCheckPrevBlock = getTimeDuration(flagSyncCheckPrevBlockInterval, sync.CheckPrevBlockInterval, "--sync-check-prevblock")
	watchInterval = getTimeDuration(flagWatchInterval, sync.WatchInterval, "--watch-interval")

	{
		if ok := common.HTTPCacheAdapterNames[flagHTTPCacheAdapter]; !ok {
			cmdcommon.PrintFlagsError(nodeCmd, "--http-cache-adapter", err)
		} else {
			httpCacheAdapter = flagHTTPCacheAdapter
		}
		var tmpUint64 uint64
		if tmpUint64, err = strconv.ParseUint(flagHTTPCachePoolSize, 10, 64); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--http-cache-pool-size", err)
		} else {
			httpCachePoolSize = int(tmpUint64)
		}
		if httpCacheAdapter == common.HTTPCacheRedisAdapterName {
			httpCacheRedisAddrs, err = parseHTTPCacheRedisAddrs(flagHTTPCacheRedisAddrs)
			if err != nil {
				cmdcommon.PrintFlagsError(nodeCmd, "--http-cache-redis-addrs", err)
			}
			if len(httpCacheRedisAddrs) <= 0 {
				err := fmt.Errorf("redis addrs is empty")
				cmdcommon.PrintFlagsError(nodeCmd, "--http-cache-redis-addrs", err)
			}
		}
	}

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--log-level", err)
	}

	var logHandler logging.Handler
	{ // global log
		var logFormatter logging.Format
		switch flagLogFormat {
		case "terminal":
			if isatty.IsTerminal(os.Stdout.Fd()) && len(flagLog) < 1 {
				logFormatter = logging.TerminalFormat()
			} else {
				logFormatter = logging.LogfmtFormat()
			}
		case "json":
			logFormatter = common.JsonFormatEx(false, true)
		default:
			cmdcommon.PrintFlagsError(nodeCmd, "--log-format", fmt.Errorf("'%s'", flagLogFormat))
		}

		if len(flagLog) < 1 {
			logHandler = logging.StreamHandler(os.Stdout, logFormatter)
		} else {
			logRotateMaxSize = parseLogRotateMaxSize("--log-rotate-max-size", flagLogRotateMaxSize)
			logRotateMaxCount = parseLogRotateMaxSize("--log-rotate-max-count", flagLogRotateMaxCount)
			logRotateMaxDays = parseLogRotateMaxDays("--log-rotate-max-days", flagLogRotateMaxDays)

			logHandler = common.NewRotateHandler(
				flagLog, logFormatter,
				logRotateMaxSize, logRotateMaxDays, logRotateMaxCount, !flagLogRotateUncompress,
			)
		}

		if logLevel == logging.LvlDebug { // only debug produces `caller` data
			logHandler = logging.CallerFileHandler(logHandler)
		}
		logHandler = logging.LvlFilterHandler(logLevel, logHandler)
		log.SetHandler(logHandler)

		runner.SetLogging(logLevel, logHandler)
		consensus.SetLogging(logLevel, logHandler)
		network.SetLogging(logLevel, logHandler)
		sync.SetLogging(logLevel, logHandler)
	}

	{ // http log
		// if without http-log, http log messages will be in `network.log`
		if len(flagHTTPLog) < 1 {
			network.SetHTTPLogging(logLevel, logHandler)
		} else {
			httpLogRotateMaxSize = parseLogRotateMaxSize("--http-log-rotate-max-size", flagHTTPLogRotateMaxSize)
			httpLogRotateMaxCount = parseLogRotateMaxSize("--http-log-rotate-max-count", flagHTTPLogRotateMaxCount)
			httpLogRotateMaxDays = parseLogRotateMaxDays("--http-log-rotate-max-days", flagHTTPLogRotateMaxDays)

			httpLogHandler := common.NewRotateHandler(
				flagHTTPLog, common.JsonFormatEx(false, true), // In `http-log`, http log will be json format
				httpLogRotateMaxSize, httpLogRotateMaxDays, httpLogRotateMaxCount, !flagHTTPLogRotateUncompress,
			)
			network.SetHTTPLogging(logging.LvlDebug, httpLogHandler) // httpLog only use `Debug`
		}
	}

	// checking `--discovery`
	l := strings.Fields(common.GetENVValue("SEBAK_DISCOVERY", ""))
	for _, i := range l {
		flagDiscovery.Set(i)
	}

	if len(flagDiscovery) < 1 {
		log.Warn("--discovery is not given; node will wait to be discovered")
	} else {
		var endpoints []*common.Endpoint
		if endpoints, err = parseFlagDiscovery(flagDiscovery); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--discovery", err)
		}
		for _, endpoint := range endpoints {
			if endpoint.Equal(publishEndpoint) {
				log.Warn(
					"--discovery is same with --publish",
					"--discovery", endpoint.String(),
					"--publish", publishEndpoint.String(),
				)
				continue
			}

			discoveryEndpoints = append(discoveryEndpoints, endpoint)
		}
	}

	if len(flagRateLimitAPI) < 1 {
		re := strings.Fields(common.GetENVValue("SEBAK_RATE_LIMIT_API", ""))
		for _, r := range re {
			flagRateLimitAPI.Set(r)
		}
	}

	rateLimitRuleAPI, err = parseFlagRateLimit(flagRateLimitAPI, common.RateLimitAPI)
	if err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--rate-limit-api", err)
	}

	if len(flagRateLimitNode) < 1 {
		re := strings.Fields(common.GetENVValue("SEBAK_RATE_LIMIT_NODE", ""))
		for _, r := range re {
			flagRateLimitNode.Set(r)
		}
	}
	rateLimitRuleNode, err = parseFlagRateLimit(flagRateLimitNode, common.RateLimitNode)
	if err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--rate-limit-node", err)
	}

	log.Info("Starting Sebak", "version", version.Version, "gitcommit", version.GitCommit)

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\tnetwork-id", flagNetworkID)
	parsedFlags = append(parsedFlags, "\n\tbind", flagBindURL)
	parsedFlags = append(parsedFlags, "\n\tjsonrpc-bind", flagJSONRPCBindURL)
	parsedFlags = append(parsedFlags, "\n\tpublish", flagPublishURL)
	parsedFlags = append(parsedFlags, "\n\tstorage", flagStorageConfigString)
	parsedFlags = append(parsedFlags, "\n\ttls-cert", flagTLSCertFile)
	parsedFlags = append(parsedFlags, "\n\ttls-key", flagTLSKeyFile)
	parsedFlags = append(parsedFlags, "\n\tlog-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "\n\tlog-format", flagLogFormat)
	parsedFlags = append(parsedFlags, "\n\tlog", flagLog)
	parsedFlags = append(parsedFlags, "\n\tthreshold", flagThreshold)
	parsedFlags = append(parsedFlags, "\n\ttimeout-init", flagTimeoutINIT)
	parsedFlags = append(parsedFlags, "\n\ttimeout-sign", flagTimeoutSIGN)
	parsedFlags = append(parsedFlags, "\n\ttimeout-accept", flagTimeoutACCEPT)
	parsedFlags = append(parsedFlags, "\n\ttimeout-allconfirm", flagTimeoutALLCONFIRM)
	parsedFlags = append(parsedFlags, "\n\tblock-time", flagBlockTime)
	parsedFlags = append(parsedFlags, "\n\tblock-time-delta", flagBlockTimeDelta)
	parsedFlags = append(parsedFlags, "\n\ttransactions-limit", flagTransactionsLimit)
	parsedFlags = append(parsedFlags, "\n\toperations-limit", flagOperationsLimit)
	parsedFlags = append(parsedFlags, "\n\toperations-in-ballot-limit", flagOperationsInBallotLimit)
	parsedFlags = append(parsedFlags, "\n\ttxpool-limit", flagTxPoolLimit)
	parsedFlags = append(parsedFlags, "\n\trate-limit-api", rateLimitRuleAPI)
	parsedFlags = append(parsedFlags, "\n\trate-limit-node", rateLimitRuleNode)
	parsedFlags = append(parsedFlags, "\n\thttp-cache-adapter", httpCacheAdapter)
	parsedFlags = append(parsedFlags, "\n\thttp-cache-pool-size", httpCachePoolSize)
	parsedFlags = append(parsedFlags, "\n\tdiscovery", discoveryEndpoints)
	parsedFlags = append(parsedFlags, "\n\twatcher-mode", flagWatcherMode)

	// create current Node
	localNode, err = node.NewLocalNode(kp, bindEndpoint, "")
	if err != nil {
		cmdcommon.PrintError(nodeCmd, err)
	}
	localNode.AddValidators(validators...)
	localNode.SetPublishEndpoint(publishEndpoint)

	var vl []interface{}
	for _, v := range localNode.GetValidators() {
		vl = append(vl, "\n\tvalidator")
		vl = append(
			vl,
			fmt.Sprintf("alias=%s address=%s endpoint=%s", v.Alias(), v.Address(), v.Endpoint()),
		)
	}
	parsedFlags = append(parsedFlags, vl...)

	log.Debug("parsed flags:", parsedFlags...)

	if flagVerbose {
		http2.VerboseLogs = true
		network.VerboseLogs = true
	}
	if flagDebugPProf {
		runner.DebugPProf = true
	}
}

func parseHTTPCacheRedisAddrs(s string) (map[string]string, error) {
	addrs := make(map[string]string)
	splitted := strings.Fields(strings.TrimSpace(s))
	for _, s := range splitted {
		addr := strings.Split(s, "=")
		if len(addr) != 2 {
			return nil, fmt.Errorf("address has wrong format")
		}
		addrs[addr[0]] = addr[1]
	}
	return addrs, nil
}

func getTimeDuration(str string, defaultValue time.Duration, errMessage string) time.Duration {
	if strings.TrimSpace(str) == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, errMessage, err)
	}
	return d
}

func runNode() error {
	metrics.InitPrometheusMetrics()
	metrics.SetVersion()

	// create network
	networkConfig, err := network.NewHTTP2NetworkConfigFromEndpoint(localNode.Alias(), bindEndpoint)
	if err != nil {
		log.Crit("failed to create network", "error", err)
		return err
	}

	nt := network.NewHTTP2Network(networkConfig)

	policy, err := consensus.NewDefaultVotingThresholdPolicy(int(threshold))
	if err != nil {
		log.Crit("failed to create VotingThresholdPolicy", "error", err)
		return err
	}

	st, err := storage.NewStorage(storageConfig)
	if err != nil {
		log.Crit("failed to initialize storage", "error", err)
		return err
	}

	// get the initial balance of geness account
	initialBalance, err := runner.GetGenesisBalance(st)
	if err != nil {
		return err
	}
	log.Debug("initial balance found", "amount", initialBalance)
	initialBalance.Invariant()

	conf := common.Config{
		TimeoutINIT:            timeoutINIT,
		TimeoutSIGN:            timeoutSIGN,
		TimeoutACCEPT:          timeoutACCEPT,
		TimeoutALLCONFIRM:      timeoutALLCONFIRM,
		NetworkID:              []byte(flagNetworkID),
		InitialBalance:         initialBalance,
		BlockTime:              blockTime,
		BlockTimeDelta:         blockTimeDelta,
		TxsLimit:               int(transactionsLimit),
		OpsLimit:               int(operationsLimit),
		OpsInBallotLimit:       int(operationsInBallotLimit),
		RateLimitRuleAPI:       rateLimitRuleAPI,
		RateLimitRuleNode:      rateLimitRuleNode,
		HTTPCacheAdapter:       httpCacheAdapter,
		HTTPCachePoolSize:      httpCachePoolSize,
		HTTPCacheRedisAddrs:    httpCacheRedisAddrs,
		CongressAccountAddress: flagCongressAddress,
		TxPoolClientLimit:      int(txPoolClientLimit),
		TxPoolNodeLimit:        int(txPoolNodeLimit),
		JSONRPCEndpoint:        jsonrpcbindEndpoint,
		WatcherMode:            flagWatcherMode,
		DiscoveryEndpoints:     discoveryEndpoints,
	}
	connectionManager := network.NewValidatorConnectionManager(localNode, nt, policy, conf)

	tp := transaction.NewPool(conf)

	c, err := sync.NewConfig(localNode, st, nt, connectionManager, tp, conf)
	if err != nil {
		return err
	}
	//Place setting config
	c.SyncPoolSize = syncPoolSize
	c.FetchTimeout = syncFetchTimeout
	c.RetryInterval = syncRetryInterval
	c.CheckBlockHeightInterval = syncCheckInterval
	c.CheckPrevBlockInterval = syncCheckPrevBlock
	c.WatchInterval = watchInterval

	syncer := c.NewSyncer()

	isaac, err := consensus.NewISAAC(localNode, policy, connectionManager, st, conf, syncer)
	if err != nil {
		log.Crit("failed to launch consensus", "error", err)
		return err
	}

	// Execution group.
	var g run.Group
	{
		nr, err := runner.NewNodeRunner(localNode, policy, nt, isaac, st, tp, conf)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return err
		}

		g.Add(func() error {
			if err := nr.Start(); err != nil {
				log.Crit("failed to start node", "error", err)
				return err
			}
			return nil
		}, func(error) {
			nr.Stop()
		})
	}
	{
		g.Add(func() error {
			return syncer.Start()
		}, func(error) {
			syncer.Stop()
		})
	}

	if flagWatcherMode {
		// In WatcherMode, get the node information from `--discovery` nodes
		localNode.ClearValidators()

		for _, endpoint := range conf.DiscoveryEndpoints {
			client := nt.GetClient(endpoint)
			if client == nil {
				err = fmt.Errorf("failed to create network client for discovery")
				log.Crit(err.Error(), "endpoint", endpoint)
				return err
			}
			var b []byte
			if b, err = client.GetNodeInfo(); err != nil {
				log.Crit("failed to get node info from discovery", "endpoint", endpoint, "error", err)
				return err
			}

			var nodeInfo node.NodeInfo
			if nodeInfo, err = node.NewNodeInfoFromJSON(b); err != nil {
				log.Crit(
					"failed to parse node info from discovery",
					"endpoint", endpoint,
					"error", err,
					"received", string(b),
				)
				return err
			}

			// Check whether basic policies are matched with remote node, like
			// `network-id`. TODO `genesis account`, `common account`, etc.
			if nodeInfo.Policy.NetworkID != string(conf.NetworkID) {
				log.Crit(
					errors.DiscoveryPolicyDoesNotMatch.Error(),
					"endpoint", endpoint,
					"remote-NetworkID", nodeInfo.Policy.NetworkID,
					"local-NetworkID", string(conf.NetworkID),
				)
				return errors.DiscoveryPolicyDoesNotMatch
			}

			if nodeInfo.Policy.InitialBalance != conf.InitialBalance {
				log.Crit(
					errors.DiscoveryPolicyDoesNotMatch.Error(),
					"endpoint", endpoint,
					"remote-InitialBalance", nodeInfo.Policy.InitialBalance,
					"local-InitialBalance", conf.InitialBalance,
				)
				return errors.DiscoveryPolicyDoesNotMatch
			}

			var validator *node.Validator
			validator, err = node.NewValidator(
				nodeInfo.Node.Address,
				nodeInfo.Node.Endpoint,
				nodeInfo.Node.Alias,
			)
			if err != nil {
				log.Crit(
					"failed to create validator from discovery",
					"endpoint", endpoint,
					"error", err,
					"node-info", nodeInfo,
				)
				return err
			}
			localNode.AddValidators(validator)
		}
		if len(localNode.GetValidators()) < 1 {
			err = fmt.Errorf("remote nodes not found")
			log.Crit(err.Error())
			return err
		}

		watcher := c.NewWatcher(syncer)
		g.Add(func() error {
			return watcher.Start()
		}, func(error) {
			watcher.Stop()
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return cmdcommon.Interrupt(cancel)
		}, func(error) {
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	return nil
}

func parseGenesisOptionFromCSV(s string) (genesisKP, commonKP keypair.KP, balance common.Amount, err error) {
	csv := strings.Split(s, ",")
	if len(csv) < 2 || len(csv) > 3 {
		err = errors.InvalidGenesisOption
		return
	}

	genesisAddress := strings.TrimSpace(csv[0])
	commonAddress := strings.TrimSpace(csv[1])
	if len(genesisAddress) < 1 || len(commonAddress) < 1 {
		err = errors.InvalidGenesisOption
		return
	}

	balanceString := common.MaximumBalance.String()
	if len(csv) == 3 {
		balanceString = strings.TrimSpace(csv[2])
	}

	return parseGenesisOption(genesisAddress, commonAddress, balanceString)
}

func parseGenesisOption(genesisAddress, commonAddress, balanceString string) (genesisKP, commonKP keypair.KP, balance common.Amount, err error) {
	if balance, err = cmdcommon.ParseAmountFromString(balanceString); err != nil {
		return
	}

	{
		if genesisKP, err = keypair.Parse(genesisAddress); err != nil {
			err = errors.NotPublicKey.Clone().SetData("error", err)
			return
		} else if _, ok := genesisKP.(*keypair.Full); ok {
			err = errors.NotPublicKey
			return
		}
	}

	{
		if commonKP, err = keypair.Parse(commonAddress); err != nil {
			err = errors.NotPublicKey.Clone().SetData("error", err)
			return
		} else if _, ok := commonKP.(*keypair.Full); ok {
			err = errors.NotPublicKey
			return
		}
	}

	return
}
