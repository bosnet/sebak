package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/http2"

	logging "github.com/inconshreveable/log15"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
)

const (
	defaultLogLevel  logging.Lvl = logging.LvlInfo
	defaultLogFormat string      = "terminal"
	defaultBindURL   string      = "https://0.0.0.0:12345"
)

var (
	flagKPSecretSeed        string = common.GetENVValue("SEBAK_SECRET_SEED", "")
	flagNetworkID           string = common.GetENVValue("SEBAK_NETWORK_ID", "")
	flagLogLevel            string = common.GetENVValue("SEBAK_LOG_LEVEL", defaultLogLevel.String())
	flagLogFormat           string = common.GetENVValue("SEBAK_LOG_FORMAT", defaultLogFormat)
	flagLog                 string = common.GetENVValue("SEBAK_LOG", "")
	flagVerbose             bool   = common.GetENVValue("SEBAK_VERBOSE", "0") == "1"
	flagBindURL             string = common.GetENVValue("SEBAK_BIND", defaultBindURL)
	flagPublishURL          string = common.GetENVValue("SEBAK_PUBLISH", "")
	flagStorageConfigString string
	flagTLSCertFile         string = common.GetENVValue("SEBAK_TLS_CERT", "sebak.crt")
	flagTLSKeyFile          string = common.GetENVValue("SEBAK_TLS_KEY", "sebak.key")
	flagValidators          string = common.GetENVValue("SEBAK_VALIDATORS", "")
	flagThreshold           string = common.GetENVValue("SEBAK_THRESHOLD", "66")
	flagTimeoutINIT         string = common.GetENVValue("SEBAK_TIMEOUT_INIT", "2")
	flagTimeoutSIGN         string = common.GetENVValue("SEBAK_TIMEOUT_SIGN", "2")
	flagTimeoutACCEPT       string = common.GetENVValue("SEBAK_TIMEOUT_ACCEPT", "2")
	flagBlockTime           string = common.GetENVValue("SEBAK_BLOCK_TIME", "5")
	flagTransactionsLimit   string = common.GetENVValue("SEBAK_TRANSACTIONS_LIMIT", "1000")
)

var (
	nodeCmd *cobra.Command

	kp                *keypair.Full
	bindEndpoint      *common.Endpoint
	publishEndpoint   *common.Endpoint
	storageConfig     *storage.Config
	validators        []*node.Validator
	threshold         int
	timeoutINIT       time.Duration
	timeoutSIGN       time.Duration
	timeoutACCEPT     time.Duration
	blockTime         time.Duration
	transactionsLimit uint64
	logLevel          logging.Lvl
	log               logging.Logger = logging.New("module", "main")
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
			if len(flagGenesis) != 0 {
				var balanceStr string
				csv := strings.Split(flagGenesis, ",")
				if len(csv) > 2 {
					cmdcommon.PrintFlagsError(nodeCmd, "--genesis",
						errors.New("--genesis expects address[,balance], but more than 2 commas detected"))
				}
				if len(csv) == 2 {
					balanceStr = csv[1]
				}
				flagName, err := MakeGenesisBlock(csv[0], flagNetworkID, balanceStr, flagStorageConfigString, log)
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
				log.Error("Node exited with error: ", err)
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
	nodeCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	nodeCmd.Flags().StringVar(&flagBindURL, "bind", flagBindURL, "bind to listen on")
	nodeCmd.Flags().StringVar(&flagPublishURL, "publish", flagPublishURL, "endpoint url for other nodes")
	nodeCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	nodeCmd.Flags().StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	nodeCmd.Flags().StringVar(&flagTLSKeyFile, "tls-key", flagTLSKeyFile, "tls key file")
	nodeCmd.Flags().StringVar(&flagValidators, "validators", flagValidators, "set validator: <endpoint url>?address=<public address>[&alias=<alias>] [ <validator>...]")
	nodeCmd.Flags().StringVar(&flagThreshold, "threshold", flagThreshold, "threshold")
	nodeCmd.Flags().StringVar(&flagTimeoutINIT, "timeout-init", flagTimeoutINIT, "timeout of the init state")
	nodeCmd.Flags().StringVar(&flagTimeoutSIGN, "timeout-sign", flagTimeoutSIGN, "timeout of the sign state")
	nodeCmd.Flags().StringVar(&flagTimeoutACCEPT, "timeout-accept", flagTimeoutACCEPT, "timeout of the accept state")
	nodeCmd.Flags().StringVar(&flagBlockTime, "block-time", flagBlockTime, "block creation time")
	nodeCmd.Flags().StringVar(&flagTransactionsLimit, "transactions-limit", flagTransactionsLimit, "transactions limit in a ballot")

	rootCmd.AddCommand(nodeCmd)
}

func parseFlagValidators(v string) (vs []*node.Validator, err error) {
	splitted := strings.Fields(v)
	if len(splitted) < 1 {
		return
	}

	for _, v := range splitted {
		var validator *node.Validator
		if validator, err = node.NewValidatorFromURI(v); err != nil {
			return
		}
		vs = append(vs, validator)
	}

	return
}

func parseFlagsNode() {
	var err error

	if len(flagNetworkID) < 1 {
		cmdcommon.PrintFlagsError(nodeCmd, "--network-id", errors.New("--network-id must be given"))
	}
	if len(flagValidators) < 1 {
		cmdcommon.PrintFlagsError(nodeCmd, "--validators", errors.New("must be given"))
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

	if len(flagPublishURL) > 0 {
		if p, err := common.ParseEndpoint(flagPublishURL); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--publish", err)
		} else {
			publishEndpoint = p
			flagPublishURL = publishEndpoint.String()
		}
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

	if validators, err = parseFlagValidators(flagValidators); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--validators", err)
	}

	for _, n := range validators {
		if n.Address() == kp.Address() {
			cmdcommon.PrintFlagsError(nodeCmd, "--validators", fmt.Errorf("duplicated public address found"))
		}
		if n.Endpoint() == bindEndpoint {
			cmdcommon.PrintFlagsError(nodeCmd, "--validators", fmt.Errorf("duplicated endpoint found"))
		}
	}

	if storageConfig, err = storage.NewConfigFromString(flagStorageConfigString); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--storage", err)
	}

	timeoutINIT = getTime(flagTimeoutINIT, 2*time.Second, "--timeout-init")
	timeoutSIGN = getTime(flagTimeoutSIGN, 2*time.Second, "--timeout-sign")
	timeoutACCEPT = getTime(flagTimeoutACCEPT, 2*time.Second, "--timeout-accept")
	blockTime = getTime(flagBlockTime, 5*time.Second, "--block-time")

	if transactionsLimit, err = strconv.ParseUint(flagTransactionsLimit, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--transactions-limit", err)
	}

	var tmpUint64 uint64
	if tmpUint64, err = strconv.ParseUint(flagThreshold, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--threshold", err)
	} else {
		threshold = int(tmpUint64)
	}

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, "--log-level", err)
	}

	var logFormatter logging.Format
	switch flagLogFormat {
	case "terminal":
		logFormatter = logging.TerminalFormat()
	case "json":
		logFormatter = common.JsonFormatEx(false, true)
	default:
		cmdcommon.PrintFlagsError(nodeCmd, "--log-format", fmt.Errorf("'%s'", flagLogFormat))
	}

	logHandler := logging.StreamHandler(os.Stdout, logFormatter)
	if len(flagLog) > 0 {
		if logHandler, err = logging.FileHandler(flagLog, logFormatter); err != nil {
			cmdcommon.PrintFlagsError(nodeCmd, "--log", err)
		}
	}

	log.SetHandler(logging.LvlFilterHandler(logLevel, logging.CallerFileHandler(logHandler)))

	runner.SetLogging(logLevel, logHandler)
	consensus.SetLogging(logLevel, logHandler)
	network.SetLogging(logLevel, logHandler)

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\tnetwork-id", flagNetworkID)
	parsedFlags = append(parsedFlags, "\n\tbind", flagBindURL)
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
	parsedFlags = append(parsedFlags, "\n\tblock-time", flagBlockTime)
	parsedFlags = append(parsedFlags, "\n\ttransactions-limit", flagTransactionsLimit)

	var vl []interface{}
	for i, v := range validators {
		vl = append(vl, fmt.Sprintf("\n\tvalidator#%d", i))
		vl = append(
			vl,
			fmt.Sprintf("alias=%s address=%s endpoint=%s", v.Alias(), v.Address(), v.Endpoint()),
		)
	}
	parsedFlags = append(parsedFlags, vl...)

	log.Debug("parsed flags:", parsedFlags...)

	if flagVerbose {
		http2.VerboseLogs = true
	}
}

func getTime(timeoutStr string, defaultValue time.Duration, errMessage string) time.Duration {
	var timeoutDuration time.Duration
	if tmpUint64, err := strconv.ParseUint(timeoutStr, 10, 64); err != nil {
		cmdcommon.PrintFlagsError(nodeCmd, errMessage, err)
	} else {
		timeoutDuration = time.Duration(tmpUint64) * time.Second
	}
	if timeoutDuration == 0 {
		timeoutDuration = defaultValue
	}
	return timeoutDuration
}

func runNode() error {
	// create current Node
	localNode, err := node.NewLocalNode(kp, bindEndpoint, "")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return err
	}
	localNode.AddValidators(validators...)
	localNode.SetPublishEndpoint(publishEndpoint)

	// create network
	networkConfig, err := network.NewHTTP2NetworkConfigFromEndpoint(localNode.Alias(), bindEndpoint)
	if err != nil {
		log.Crit("failed to create network", "error", err)
		return err
	}

	nt := network.NewHTTP2Network(networkConfig)

	policy, err := consensus.NewDefaultVotingThresholdPolicy(threshold, threshold)
	if err != nil {
		log.Crit("failed to create VotingThresholdPolicy", "error", err)
		return err
	}

	connectionManager := network.NewValidatorConnectionManager(
		localNode,
		nt,
		policy,
	)

	isaac, err := consensus.NewISAAC([]byte(flagNetworkID), localNode, policy, connectionManager)
	if err != nil {
		log.Crit("failed to launch consensus", "error", err)
		return err
	}

	st, err := storage.NewStorage(storageConfig)
	if err != nil {
		log.Crit("failed to initialize storage", "error", err)
		return err
	}

	// Execution group.
	var g run.Group
	{
		conf := &consensus.ISAACConfiguration{
			TimeoutINIT:       timeoutINIT,
			TimeoutSIGN:       timeoutSIGN,
			TimeoutACCEPT:     timeoutACCEPT,
			BlockTime:         blockTime,
			TransactionsLimit: uint64(transactionsLimit),
		}
		nr, err := runner.NewNodeRunner(flagNetworkID, localNode, policy, nt, isaac, st, conf)

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
