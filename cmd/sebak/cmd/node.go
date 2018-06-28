package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/http2"

	logging "github.com/inconshreveable/log15"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
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
	flagSignThreshold       string = sebakcommon.GetENVValue("SEBAK_SIGN_THRESHOLD", "60")
	flagAcceptThreshold     string = sebakcommon.GetENVValue("SEBAK_ACCEPT_THRESHOLD", "60")
)

var (
	nodeCmd *cobra.Command

	kp            *keypair.Full
	nodeEndpoint  *sebakcommon.Endpoint
	storageConfig *sebakstorage.Config
	validators    []*sebakcommon.Validator
	logLevel      logging.Lvl
	log           logging.Logger
)

func init() {
	var err error

	nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run sebak node",
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
	nodeCmd.Flags().StringVar(&flagEndpointString, "endpoint", flagEndpointString, "endpoint uri to listen on")
	nodeCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	nodeCmd.Flags().StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	nodeCmd.Flags().StringVar(&flagTLSKeyFile, "tls-key", flagTLSKeyFile, "tls key file")
	nodeCmd.Flags().StringVar(&flagValidators, "validators", flagValidators, "set validator: <endpoint url>?address=<public address>[&alias=<alias>] [ <validator>...]")
	nodeCmd.Flags().StringVar(&flagSignThreshold, "sign-threshold", flagSignThreshold, "sign threshold")
	nodeCmd.Flags().StringVar(&flagAcceptThreshold, "accept-threshold", flagAcceptThreshold, "accept threshold")

	rootCmd.AddCommand(nodeCmd)
}

func parseFlagValidators(v string) (vs []*sebakcommon.Validator, err error) {
	splitted := strings.Fields(v)
	if len(splitted) < 1 {
		return
	}

	for _, v := range splitted {
		var validator *sebakcommon.Validator
		if validator, err = sebakcommon.NewValidatorFromURI(v); err != nil {
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

	if p, err := sebakcommon.ParseNodeEndpoint(flagEndpointString); err != nil {
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
	queries.Add("NodeName", sebakcommon.MakeAlias(kp.Address()))
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
	parsedFlags = append(parsedFlags, "\n\tsign-threshold", flagSignThreshold)
	parsedFlags = append(parsedFlags, "\n\taccept-threshold", flagAcceptThreshold)

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

func runNode() {
	// create current Node
	currentNode, err := sebakcommon.NewValidator(kp.Address(), nodeEndpoint, "")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return
	}
	currentNode.SetKeypair(kp)
	currentNode.AddValidators(validators...)

	// create network
	nt, err := sebaknetwork.NewNetwork(nodeEndpoint)
	if err != nil {
		log.Crit("transport error", "error", err)

		os.Exit(1)
	}

	signTh, err := strconv.Atoi(flagSignThreshold)
	acceptTh, err := strconv.Atoi(flagAcceptThreshold)
	policy, _ := sebak.NewDefaultVotingThresholdPolicy(100, signTh, acceptTh)
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

	// Execution group.
	var g run.Group
	{
		nr := sebak.NewNodeRunner(flagNetworkID, currentNode, policy, nt, isaac, st)
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
		os.Exit(1)
	}
}
