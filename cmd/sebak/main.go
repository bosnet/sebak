package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	logging "github.com/inconshreveable/log15"
	"github.com/mattn/go-isatty"
	"github.com/stellar/go/keypair"
	"golang.org/x/net/http2"

	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/storage"
)

// TODO "github.com/cockroachdb/cmux", split request streams
// TODO "github.com/spf13/cobra", cli commands and options

const defaultNetwork string = "https"
const defaultPort int = 12345
const defaultHost string = "0.0.0.0"
const defaultLogLevel logging.Lvl = logging.LvlInfo

type FlagValidators []*sebakcommon.Validator

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

var (
	flags *flag.FlagSet

	kp                 *keypair.Full
	flagKPSecretSeed   string = sebakcommon.GetENVValue("SEBAK_SECRET_SEED", "")
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

	nodeEndpoint  *sebakcommon.Endpoint
	storageConfig *storage.Config
	logLevel      logging.Lvl
	log           logging.Logger
)

func printFlagsError(flagName string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid '%s'; %v\n\n", flagName, err)
	}

	flags.Usage()

	os.Exit(1)
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var err error

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options]")

		flags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	// storage
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		printFlagsError("-tls-cert", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		printFlagsError("-tls-cert", err)
	}
	flagStorageConfigString = sebakcommon.GetENVValue("SEBAK_STORAGE", fmt.Sprintf("file://%s/db", currentDirectory))

	// flags
	flags.StringVar(&flagKPSecretSeed, "secret-seed", flagKPSecretSeed, "secret seed of this node")
	flags.StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	flags.StringVar(&flagLogOutput, "log-output", flagLogOutput, "set log output file")
	flags.BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	flags.StringVar(&flagEndpointString, "endpoint", flagEndpointString, "endpoint uri to listen on ('https://0.0.0.0:12345')")
	flags.StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	flags.StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	flags.StringVar(&flagTLSKeyFile, "tls-Key", flagTLSKeyFile, "tls Keyificate file")
	flags.Var(&flagValidators, "validator", "set validator: '<public address>,<endpoint url>,<alias>' or <public address>,<endpoint url>")

	flags.Parse(os.Args[1:])

	if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
		printFlagsError("-tls-cert", err)
	}
	if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
		printFlagsError("-tls-key", err)
	}

	var parsedKP keypair.KP
	parsedKP, err = keypair.Parse(flagKPSecretSeed)
	if err != nil {
		printFlagsError("-secret-seed", err)
	} else {
		kp = parsedKP.(*keypair.Full)
	}

	if p, err := sebakcommon.ParseNodeEndpoint(flagEndpointString); err != nil {
		printFlagsError("-endpoint", err)
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
			printFlagsError("-validator", fmt.Errorf("duplicated public address found"))
		}
		if n.Endpoint() == nodeEndpoint {
			printFlagsError("-validator", fmt.Errorf("duplicated endpoint found"))
		}
	}

	if storageConfig, err = storage.NewConfigFromString(flagStorageConfigString); err != nil {
		printFlagsError("-storage", err)
	}

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		printFlagsError("-log-level", err)
	}

	var logHandler logging.Handler

	var formatter logging.Format
	if isatty.IsTerminal(os.Stdout.Fd()) {
		formatter = logging.TerminalFormat()
	} else {
		formatter = logging.JsonFormatEx(false, true)
	}
	logHandler = logging.StreamHandler(os.Stdout, formatter)

	if len(flagLogOutput) > 0 {
		if logHandler, err = logging.FileHandler(flagLogOutput, logging.JsonFormat()); err != nil {
			printFlagsError("-log-output", err)
		}
	}

	log = logging.New("module", "main")
	log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	sebak.SetLogging(logLevel, logHandler)
	sebaknetwork.SetLogging(logLevel, logHandler)

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\tlog-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "\n\tlog-output", flagLogOutput)
	parsedFlags = append(parsedFlags, "\n\tendpoint", flagEndpointString)
	parsedFlags = append(parsedFlags, "\n\tstorage", flagStorageConfigString)
	parsedFlags = append(parsedFlags, "\n\ttls-cert", flagTLSCertFile)
	parsedFlags = append(parsedFlags, "\n\ttls-key", flagTLSKeyFile)

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

func main() {
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

	isaac, err := sebak.NewISAAC(currentNode, policy)
	if err != nil {
		log.Error("failed to launch consensus", "error", err)
		return
	}

	st, err := storage.NewStorage(storageConfig)
	if err != nil {
		log.Crit("failed to initialize storage", "error", err)

		os.Exit(1)
	}
	nr := sebak.NewNodeRunner(currentNode, policy, nt, isaac, st)
	if err := nr.Start(); err != nil {
		log.Crit("failed to start node", "error", err)

		os.Exit(1)
	}
}
