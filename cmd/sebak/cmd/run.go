package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/http2"

	logging "github.com/inconshreveable/log15"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"boscoin.io/sebak/cmd/sebak/common"

	"strconv"
)

const defaultNetwork string = "https"
const defaultPort int = 12345
const defaultHost string = "0.0.0.0"
const defaultLogLevel logging.Lvl = logging.LvlInfo

var (
	flagKPSecretSeed   string = sebakcommon.GetENVValue("SEBAK_SECRET_SEED", "")
	flagNetworkID      string = sebakcommon.GetENVValue("SEBAK_NETWORK_ID", "")
	flagLogLevel       string = sebakcommon.GetENVValue("SEBAK_LOG_LEVEL", defaultLogLevel.String())
	flagLogOutput      string = sebakcommon.GetENVValue("SEBAK_LOG_OUTPUT", "")
	flagVerbose        bool   = sebakcommon.GetENVValue("SEBAK_VERBOSE", "0") == "1"
	flagEndpointString string = sebakcommon.GetENVValue(
		"SEBAK_ENDPOINT",
		fmt.Sprintf("%s://%s:%d", defaultNetwork, defaultHost, defaultPort),
	)
	flagStorageConfigString string
	flagTLSCertFile         string = sebakcommon.GetENVValue("SEBAK_TLS_CERT", "sebak.crt")
	flagTLSKeyFile          string = sebakcommon.GetENVValue("SEBAK_TLS_KEY", "sebak.key")
	flagValidators          string = sebakcommon.GetENVValue("SEBAK_VALIDATORS", "")
	flagThreshold           string = sebakcommon.GetENVValue("SEBAK_THRESHOLD", "66")
	flagTimeoutINIT         string = sebakcommon.GetENVValue("SEBAK_TIMEOUT_INIT", "2")
	flagTimeoutSIGN         string = sebakcommon.GetENVValue("SEBAK_TIMEOUT_SIGN", "2")
	flagTimeoutACCEPT       string = sebakcommon.GetENVValue("SEBAK_TIMEOUT_ACCEPT", "2")
	flagTimeoutALLCONFIRM   string = sebakcommon.GetENVValue("SEBAK_TIMEOUT_ALLCONFIRM", "2")
	flagTransactionsLimit   string = sebakcommon.GetENVValue("SEBAK_TRANSACTIONS_LIMIT", "1000")
)

var (
	nodeCmd *cobra.Command

	kp                *keypair.Full
	nodeEndpoint      *sebakcommon.Endpoint
	storageConfig     *sebakstorage.Config
	validators        []*sebaknode.Validator
	threshold         int
	timeoutINIT       time.Duration
	timeoutSIGN       time.Duration
	timeoutACCEPT     time.Duration
	timeoutALLCONFIRM time.Duration
	transactionsLimit uint64
	logLevel          logging.Lvl
	log               logging.Logger
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
					common.PrintFlagsError(nodeCmd, "--genesis",
						errors.New("--genesis expects address[,balance], but more than 2 commas detected"))
				}
				if len(csv) == 2 {
					balanceStr = csv[1]
				}
				flagName, err := MakeGenesisBlock(csv[0], flagNetworkID, balanceStr, flagStorageConfigString)
				if len(flagName) != 0 || err != nil {
					common.PrintFlagsError(c, flagName, err)
				}
			}

			parseFlagsNode()

			if err = runNode(); err != nil {
				// TODO: Handle errors here
				// We should handle the error correctly, however since we don't currently
				// shut down the HTTP server correctly, we get an error,
				// and we need the binary to exit with a successfull error code for
				// code coverage in integration test to work.
				log.Error("Node exited with error: %v", err)
			}
		},
	}

	// storage
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		common.PrintFlagsError(nodeCmd, "-tls-cert", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		common.PrintFlagsError(nodeCmd, "-tls-cert", err)
	}
	flagStorageConfigString = sebakcommon.GetENVValue("SEBAK_STORAGE", fmt.Sprintf("file://%s/db", currentDirectory))

	nodeCmd.Flags().StringVar(&flagGenesis, "genesis", flagGenesis, "performs the 'genesis' command before running node. Syntax: key[,balance]")
	nodeCmd.Flags().StringVar(&flagKPSecretSeed, "secret-seed", flagKPSecretSeed, "secret seed of this node")
	nodeCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	nodeCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	nodeCmd.Flags().StringVar(&flagLogOutput, "log-output", flagLogOutput, "set log output file")
	nodeCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	nodeCmd.Flags().StringVar(&flagEndpointString, "endpoint", flagEndpointString, "endpoint uri to listen on")
	nodeCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	nodeCmd.Flags().StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	nodeCmd.Flags().StringVar(&flagTLSKeyFile, "tls-key", flagTLSKeyFile, "tls key file")
	nodeCmd.Flags().StringVar(&flagValidators, "validators", flagValidators, "set validator: <endpoint url>?address=<public address>[&alias=<alias>] [ <validator>...]")
	nodeCmd.Flags().StringVar(&flagThreshold, "threshold", flagThreshold, "threshold")
	nodeCmd.Flags().StringVar(&flagTimeoutINIT, "timeout-init", flagTimeoutINIT, "timeout of the init state")
	nodeCmd.Flags().StringVar(&flagTimeoutSIGN, "timeout-sign", flagTimeoutSIGN, "timeout of the sign state")
	nodeCmd.Flags().StringVar(&flagTimeoutACCEPT, "timeout-accept", flagTimeoutACCEPT, "timeout of the accept state")
	nodeCmd.Flags().StringVar(&flagTimeoutALLCONFIRM, "timeout-allconfirm", flagTimeoutALLCONFIRM, "timeout of the allconfirm state")
	nodeCmd.Flags().StringVar(&flagTransactionsLimit, "transactions-limit", flagTransactionsLimit, "transactions limit in a ballot")

	rootCmd.AddCommand(nodeCmd)
}

func parseFlagValidators(v string) (vs []*sebaknode.Validator, err error) {
	splitted := strings.Fields(v)
	if len(splitted) < 1 {
		return
	}

	for _, v := range splitted {
		var validator *sebaknode.Validator
		if validator, err = sebaknode.NewValidatorFromURI(v); err != nil {
			return
		}
		vs = append(vs, validator)
	}

	return
}

func parseFlagsNode() {
	var err error

	if len(flagNetworkID) < 1 {
		common.PrintFlagsError(nodeCmd, "--network-id", errors.New("--network-id must be given"))
	}
	if len(flagValidators) < 1 {
		common.PrintFlagsError(nodeCmd, "--validators", errors.New("must be given"))
	}
	if len(flagKPSecretSeed) < 1 {
		common.PrintFlagsError(nodeCmd, "--secret-seed", errors.New("must be given"))
	}

	var parsedKP keypair.KP
	parsedKP, err = keypair.Parse(flagKPSecretSeed)
	if err != nil {
		common.PrintFlagsError(nodeCmd, "--secret-seed", err)
	} else {
		kp = parsedKP.(*keypair.Full)
	}

	if p, err := sebakcommon.ParseEndpoint(flagEndpointString); err != nil {
		common.PrintFlagsError(nodeCmd, "--endpoint", err)
	} else {
		nodeEndpoint = p
		flagEndpointString = nodeEndpoint.String()
	}

	if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
		common.PrintFlagsError(nodeCmd, "--tls-cert", err)
	}
	if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
		common.PrintFlagsError(nodeCmd, "--tls-key", err)
	}

	queries := nodeEndpoint.Query()
	queries.Add("TLSCertFile", flagTLSCertFile)
	queries.Add("TLSKeyFile", flagTLSKeyFile)
	queries.Add("IdleTimeout", "3s")
	queries.Add("NodeName", sebaknode.MakeAlias(kp.Address()))
	nodeEndpoint.RawQuery = queries.Encode()

	if validators, err = parseFlagValidators(flagValidators); err != nil {
		common.PrintFlagsError(nodeCmd, "--validators", err)
	}

	for _, n := range validators {
		if n.Address() == kp.Address() {
			common.PrintFlagsError(nodeCmd, "--validator", fmt.Errorf("duplicated public address found"))
		}
		if n.Endpoint() == nodeEndpoint {
			common.PrintFlagsError(nodeCmd, "--validator", fmt.Errorf("duplicated endpoint found"))
		}
	}

	if storageConfig, err = sebakstorage.NewConfigFromString(flagStorageConfigString); err != nil {
		common.PrintFlagsError(nodeCmd, "--storage", err)
	}

	timeoutINIT = getTimeout(flagTimeoutINIT, "--timeout-init")
	timeoutSIGN = getTimeout(flagTimeoutSIGN, "--timeout-sign")
	timeoutACCEPT = getTimeout(flagTimeoutACCEPT, "--timeout-accept")
	timeoutALLCONFIRM = getTimeout(flagTimeoutALLCONFIRM, "--timeout-allconfirm")

	if transactionsLimit, err = strconv.ParseUint(flagTransactionsLimit, 10, 64); err != nil {
		common.PrintFlagsError(nodeCmd, "--transactions-limit", err)
	}

	var tmpUint64 uint64
	if tmpUint64, err = strconv.ParseUint(flagThreshold, 10, 64); err != nil {
		common.PrintFlagsError(nodeCmd, "--threshold", err)
	} else {
		threshold = int(tmpUint64)
	}

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		common.PrintFlagsError(nodeCmd, "--log-level", err)
	}

	logHandler := logging.StdoutHandler

	if len(flagLogOutput) < 1 {
		flagLogOutput = "<stdout>"
	} else {
		if logHandler, err = logging.FileHandler(flagLogOutput, logging.JsonFormat()); err != nil {
			common.PrintFlagsError(nodeCmd, "--log-output", err)
		}
	}

	logHandler = logging.CallerFileHandler(logHandler)

	log = logging.New("module", "main")
	log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	sebak.SetLogging(logLevel, logHandler)
	sebaknetwork.SetLogging(logLevel, logHandler)

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\tnetwork-id", flagNetworkID)
	parsedFlags = append(parsedFlags, "\n\tendpoint", flagEndpointString)
	parsedFlags = append(parsedFlags, "\n\tstorage", flagStorageConfigString)
	parsedFlags = append(parsedFlags, "\n\ttls-cert", flagTLSCertFile)
	parsedFlags = append(parsedFlags, "\n\ttls-key", flagTLSKeyFile)
	parsedFlags = append(parsedFlags, "\n\tlog-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "\n\tlog-output", flagLogOutput)
	parsedFlags = append(parsedFlags, "\n\tthreshold", flagThreshold)
	parsedFlags = append(parsedFlags, "\n\ttimeout-init", flagTimeoutINIT)
	parsedFlags = append(parsedFlags, "\n\ttimeout-sign", flagTimeoutSIGN)
	parsedFlags = append(parsedFlags, "\n\ttimeout-accept", flagTimeoutACCEPT)
	parsedFlags = append(parsedFlags, "\n\ttimeout-allconfirm", flagTimeoutALLCONFIRM)
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

func getTimeout(timeoutStr string, errMessage string) time.Duration {
	var timeoutDuration time.Duration
	if tmpUint64, err := strconv.ParseUint(flagTimeoutINIT, 10, 64); err != nil {
		common.PrintFlagsError(nodeCmd, errMessage, err)
	} else {
		timeoutDuration = time.Duration(tmpUint64) * time.Second
	}
	if timeoutDuration == 0 {
		timeoutDuration = 2 * time.Second
	}
	return timeoutDuration
}

func runNode() error {
	// create current Node
	localNode, err := sebaknode.NewLocalNode(kp, nodeEndpoint, "")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return err
	}
	localNode.AddValidators(validators...)

	// create network
	nt, err := sebaknetwork.NewNetwork(nodeEndpoint)
	if err != nil {
		log.Crit("failed to create Network", "error", err)
		return err
	}

	policy, err := sebak.NewDefaultVotingThresholdPolicy(threshold, threshold)
	if err != nil {
		log.Crit("failed to create VotingThresholdPolicy", "error", err)
		return err
	}

	isaac, err := sebak.NewISAAC([]byte(flagNetworkID), localNode, policy)
	if err != nil {
		log.Crit("failed to launch consensus", "error", err)
		return err
	}

	st, err := sebakstorage.NewStorage(storageConfig)
	if err != nil {
		log.Crit("failed to initialize storage", "error", err)
		return err
	}

	// Execution group.
	var g run.Group
	{
		nr, err := sebak.NewNodeRunner(flagNetworkID, localNode, policy, nt, isaac, st)
		conf := &sebak.IsaacConfiguration{
			TimeoutINIT:       timeoutINIT,
			TimeoutSIGN:       timeoutSIGN,
			TimeoutACCEPT:     timeoutACCEPT,
			TimeoutALLCONFIRM: timeoutALLCONFIRM,
			TransactionsLimit: uint64(transactionsLimit),
		}
		nr.SetConf(conf)

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
			return common.Interrupt(cancel)
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
