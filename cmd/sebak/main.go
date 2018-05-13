package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/net/http2"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

// TODO "github.com/cockroachdb/cmux", split request streams
// TODO "github.com/spf13/cobra", cli commands and options

const defaultNetwork string = "https"
const defaultPort int = 12345
const defaultHost string = "0.0.0.0"
const defaultLogLevel logging.Lvl = logging.LvlInfo

type FlagValidators []*util.Validator

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

	endpoint, err := util.ParseNodeEndpoint(parsed[1])
	if err != nil {
		return err
	}
	node, err := util.NewValidator(parsed[0], endpoint, parsed[2])
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
	flagKPSecretSeed   string = util.GetENVValue("SEBAK_SECRET_SEED", "")
	flagLogLevel       string = util.GetENVValue("SEBAK_LOG_LEVEL", defaultLogLevel.String())
	flagLogOutput      string = util.GetENVValue("SEBAK_LOG_OUTPUT", "")
	flagVerbose        bool   = false
	nodeEndpoint       *util.Endpoint
	flagEndpointString string = util.GetENVValue(
		"SEBAK_ENDPOINT",
		fmt.Sprintf("%s://%s:%d", defaultNetwork, defaultHost, defaultPort),
	)
	flagTLSCertFile string = util.GetENVValue("SEBAK_TLS_CERT", "sebak.crt")
	flagTLSKeyFile  string = util.GetENVValue("SEBAK_TLS_KEY", "sebak.key")
	flagValidators  FlagValidators

	logLevel logging.Lvl
	log      logging.Logger
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

		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flagVerbose = util.GetENVValue("SEBAK_VERBOSE", "0") == "1"

	// flags
	flags.StringVar(&flagKPSecretSeed, "secret-seed", flagKPSecretSeed, "secret seed of this node")
	flags.StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	flags.StringVar(&flagLogOutput, "log-output", flagLogOutput, "set log output file")
	flags.BoolVar(&flagVerbose, "verbose", flagVerbose, "verbose")
	flags.StringVar(&flagEndpointString, "endpoint", flagEndpointString, "endpoint uri to listen on ('https://0.0.0.0:12345')")
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

	if p, err := util.ParseNodeEndpoint(flagEndpointString); err != nil {
		printFlagsError("-endpoint", err)
	} else {
		nodeEndpoint = p
		flagEndpointString = nodeEndpoint.String()
	}

	queries := url.Values{}
	queries.Add("TLSCertFile", flagTLSCertFile)
	queries.Add("TLSKeyFile", flagTLSKeyFile)
	queries.Add("IdleTimeout", "3s")
	nodeEndpoint.RawQuery = queries.Encode()

	for _, n := range flagValidators {
		if n.Address() == kp.Address() {
			printFlagsError("-validator", fmt.Errorf("duplicated public address found"))
			break
		}
		if n.Endpoint() == nodeEndpoint {
			printFlagsError("-validator", fmt.Errorf("duplicated endpoint found"))
			break
		}
	}

	if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
		printFlagsError("-log-level", err)
	}

	var logHandler logging.Handler
	logHandler = logging.StreamHandler(os.Stdout, logging.TerminalFormat())
	if len(flagLogOutput) > 0 {
		if logHandler, err = logging.FileHandler(flagLogOutput, logging.JsonFormat()); err != nil {
			printFlagsError("-log-output", err)
		}
	}

	log = logging.New("module", "main")
	log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	sebak.SetLogging(logLevel, logHandler)

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "\n\tlog-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "\n\tlog-output", flagLogOutput)
	parsedFlags = append(parsedFlags, "\n\tendpoint", flagEndpointString)
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

	// NOTE instead of set `http2.VerboseLogs`, just use
	// `GODEBUG="http2debug=2"`.
	if flagVerbose {
		http2.VerboseLogs = true
	}
}

func main() {
	// create current Node
	currentNode, err := util.NewValidator(kp.Address(), nodeEndpoint, "self")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return
	}
	currentNode.SetKeypair(kp)
	currentNode.AddValidators(flagValidators...)

	// create network
	nt, err := network.NewTransportServer(nodeEndpoint)
	if err != nil {
		log.Crit("transport error", "error", err)

		os.Exit(1)
	}

	// TODO policy threshold can be set in cmd options
	policy, _ := sebak.NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(uint64(len(currentNode.GetValidators())) + 1) // including 'self'

	isaac, err := sebak.NewISAAC(currentNode, policy)
	if err != nil {
		log.Error("failed to launch consensus", "error", err)
		return
	}

	nr := sebak.NewNodeRunner(currentNode, policy, nt, isaac)
	nr.Ready()

	if err := nr.Start(); err != nil {
		log.Crit("failed to start node", "error", err)

		os.Exit(1)
	}
}

func main0() {
	// create current Node
	currentNode, err := util.NewValidator(kp.Address(), nodeEndpoint, "self")
	if err != nil {
		log.Error("failed to launch main node", "error", err)
		return
	}
	currentNode.SetKeypair(kp)
	currentNode.AddValidators(flagValidators...)

	// create network
	//ctx := context.WithValue(context.Background(), "currentNode", currentNode)
	nt, err := network.NewTransportServer(nodeEndpoint)
	if err != nil {
		log.Crit("transport error", "error", err)

		os.Exit(1)
	}

	go func() {
		nt.Start()
	}()

	nt.Ready()

	// TODO policy threshold can be set in cmd options
	policy, _ := sebak.NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(uint64(len(currentNode.GetValidators())) + 1) // including 'self'

	is, err := sebak.NewISAAC(currentNode, policy)
	if err != nil {
		log.Error("failed to launch consensus", "error", err)
		return
	}

	for message := range nt.ReceiveMessage() {
		log.Debug("got message", "message", message)

		switch message.Type {
		case "message":
			var tx sebak.Transaction
			if tx, err = sebak.NewTransactionFromJSON(message.Data); err != nil {
				log.Error("found invalid transaction message", "error", err)

				// TODO if failed, save in `BlockTransactionHistory`????
				continue
			}
			if err = tx.IsWellFormed(); err != nil {
				log.Error("found invalid transaction message", "error", err)
				// TODO if failed, save in `BlockTransactionHistory`
				continue
			}

			/*
				- TODO `Message` must be saved in `BlockTransactionHistory`
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			var ballot sebak.Ballot
			if ballot, err = is.ReceiveMessage(tx); err != nil {
				log.Error("failed to receive new message", "error", err)
				continue
			}

			// TODO initially shutup and broadcast
			fmt.Println(ballot)
		case "ballot":
			/*
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			var ballot sebak.Ballot
			if ballot, err = sebak.NewBallotFromJSON(message.Data); err != nil {
				log.Error("found invalid ballot message", "error", err)
				continue
			}
			var vt sebak.VotingStateStaging
			if vt, err = is.ReceiveBallot(ballot); err != nil {
				log.Error("failed to receive ballot", "error", err)
				continue
			}

			if vt.IsEmpty() {
				continue
			}

			if vt.IsClosed() {
				if !vt.IsStorable() {
					continue
				}
				// store in BlockTransaction
			}

			if !vt.IsChanged() {
				continue
			}

			// TODO state is changed, so broadcast

			fmt.Println(vt)
		}
	}

	select {}

	os.Exit(0)
}
