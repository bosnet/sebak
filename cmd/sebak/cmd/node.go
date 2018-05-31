package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/http2"

	logging "github.com/inconshreveable/log15"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/storage"

	"github.com/spikeekips/sebak/cmd/sebak/common"
)

type FlagValidators []*sebakcommon.Validator

func (f *FlagValidators) Type() string {
	return "validators"
}

func (f *FlagValidators) String() string {
	return ""
}

func (f *FlagValidators) Set(v string) error {
	if strings.Count(v, ",") > 2 {
		return errors.New("multiple comma, ',' found")
	}

	parsed := strings.SplitN(v, ",", 3)
	if len(parsed) < 2 {
		return errors.New("at least '<public address>,<endpoint url>' must be given")
	}
	if len(parsed) < 3 {
		parsed = append(parsed, "")
	}

	endpoint, err := sebakcommon.ParseNodeEndpoint(parsed[1])
	if err != nil {
		return err
	}
	node, err := sebakcommon.NewValidator(parsed[0], endpoint, parsed[2])
	if err != nil {
		return fmt.Errorf("failed to create validator: %v", err)
	}

	// check duplication
	for _, n := range *f {
		if node.Address() == n.Address() {
			return fmt.Errorf("duplicated public address found")
		}
		if node.Endpoint() == n.Endpoint() {
			return fmt.Errorf("duplicated endpoint found")
		}
	}

	*f = append(*f, node)

	return nil
}

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
	flagValidators          FlagValidators
)

var (
	nodeCmd *cobra.Command

	kp            *keypair.Full
	nodeEndpoint  *sebakcommon.Endpoint
	storageConfig *sebakstorage.Config
	logLevel      logging.Lvl
	log           logging.Logger
)

func init() {
	var err error

	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run sekbak node",
		Run: func(c *cobra.Command, args []string) {
			parseFlagsNode()

			runNode()
			return
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

	nodeCmd.Flags().StringVar(&flagKPSecretSeed, "secret-seed", flagKPSecretSeed, "secret seed of this node")
	nodeCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	nodeCmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	nodeCmd.Flags().StringVar(&flagLogOutput, "log-output", flagLogOutput, "set log output file")
	nodeCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	nodeCmd.Flags().StringVar(&flagEndpointString, "endpoint", flagEndpointString, "endpoint uri to listen on ('https://0.0.0.0:12345')")
	nodeCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	nodeCmd.Flags().StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	nodeCmd.Flags().StringVar(&flagTLSKeyFile, "tls-Key", flagTLSKeyFile, "tls Keyificate file")
	nodeCmd.Flags().Var(&flagValidators, "validator", "set validator: '<public address>,<endpoint url>,<alias>' or <public address>,<endpoint url>")

	nodeCmd.MarkFlagRequired("network-id")
	nodeCmd.MarkFlagRequired("secret-seed")
	nodeCmd.MarkFlagRequired("validator")

	rootCmd.AddCommand(nodeCmd)
}

func parseFlagsNode() {
	var err error

	if len(flagNetworkID) < 1 {
		common.PrintFlagsError(nodeCmd, "--network-id", errors.New("-network-id must be given"))
	}

	if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
		common.PrintFlagsError(nodeCmd, "--tls-cert", err)
	}
	if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
		common.PrintFlagsError(nodeCmd, "--tls-key", err)
	}

	var parsedKP keypair.KP
	parsedKP, err = keypair.Parse(flagKPSecretSeed)
	if err != nil {
		common.PrintFlagsError(nodeCmd, "--secret-seed", err)
	} else {
		kp = parsedKP.(*keypair.Full)
	}

	if p, err := sebakcommon.ParseNodeEndpoint(flagEndpointString); err != nil {
		common.PrintFlagsError(nodeCmd, "--endpoint", err)
	} else {
		nodeEndpoint = p
		flagEndpointString = nodeEndpoint.String()
	}

	queries := nodeEndpoint.Query()
	queries.Add("TLSCertFile", flagTLSCertFile)
	queries.Add("TLSKeyFile", flagTLSKeyFile)
	queries.Add("IdleTimeout", "3s")
	queries.Add("NodeName", sebakcommon.MakeAlias(kp.Address()))
	nodeEndpoint.RawQuery = queries.Encode()

	for _, n := range flagValidators {
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

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		common.PrintFlagsError(nodeCmd, "--log-level", err)
	}

	var logHandler logging.Handler

	var formatter logging.Format
	if isatty.IsTerminal(os.Stdout.Fd()) {
		formatter = logging.TerminalFormat()
	} else {
		formatter = logging.JsonFormatEx(false, true)
	}
	logHandler = logging.StreamHandler(os.Stdout, formatter)

	if len(flagLogOutput) < 1 {
		flagLogOutput = "<stdout>"
	} else {
		if logHandler, err = logging.FileHandler(flagLogOutput, logging.JsonFormat()); err != nil {
			common.PrintFlagsError(nodeCmd, "--log-output", err)
		}
	}

	log = logging.New("module", "main")
	log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	sebak.SetLogging(logLevel, logHandler)
	sebaknetwork.SetLogging(logLevel, logHandler)

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\network-id", flagNetworkID)
	parsedFlags = append(parsedFlags, "\n\tendpoint", flagEndpointString)
	parsedFlags = append(parsedFlags, "\n\tstorage", flagStorageConfigString)
	parsedFlags = append(parsedFlags, "\n\ttls-cert", flagTLSCertFile)
	parsedFlags = append(parsedFlags, "\n\ttls-key", flagTLSKeyFile)
	parsedFlags = append(parsedFlags, "\n\tlog-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "\n\tlog-output", flagLogOutput)

	var vl []interface{}
	for i, v := range flagValidators {
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

func runNode() {
	// create current Node
	currentNode, err := sebakcommon.NewValidator(kp.Address(), nodeEndpoint, "")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return
	}
	currentNode.SetKeypair(kp)
	currentNode.AddValidators(flagValidators...)

	// create network
	nt, err := sebaknetwork.NewNetwork(nodeEndpoint)
	if err != nil {
		log.Crit("transport error", "error", err)

		os.Exit(1)
	}

	// TODO policy threshold can be set in cmd options
	policy, _ := sebak.NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(len(currentNode.GetValidators()) + 1) // including 'self'

	isaac, err := sebak.NewISAAC([]byte(flagNetworkID), currentNode, policy)
	if err != nil {
		log.Error("failed to launch consensus", "error", err)
		return
	}

	st, err := sebakstorage.NewStorage(storageConfig)
	if err != nil {
		log.Crit("failed to initialize storage", "error", err)

		os.Exit(1)
	}
	nr := sebak.NewNodeRunner(flagNetworkID, currentNode, policy, nt, isaac, st)
	if err := nr.Start(); err != nil {
		log.Crit("failed to start node", "error", err)

		os.Exit(1)
	}
}
