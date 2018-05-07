package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
)

// TODO "github.com/cockroachdb/cmux", split request streams
// TODO "github.com/spf13/cobra", cli commands and options

const defaultPort int = 12345
const defaultHost string = "localhost"
const defaultLogLevel logging.Lvl = logging.LvlInfo

var (
	flags *flag.FlagSet

	flagKPSecretSeed    string = util.GetENVValue("SEBAK_secret_seed", "")
	flagKPPublicAddress string = util.GetENVValue("SEBAK_secret_seed", "")
	flagLogLevel        string = util.GetENVValue("SEBAK_LOG_LEVEL", defaultLogLevel.String())
	flagLogOutput       string = util.GetENVValue("SEBAK_LOG_OUTPUT", "")
	flagBind            string = util.GetENVValue("SEBAK_BIND", fmt.Sprintf("%s:%d", defaultHost, defaultPort))
	flagTLSCertFile     string = util.GetENVValue("SEBAK_TLS_CERT", "sebak.crt")
	flagTLSKeyFile      string = util.GetENVValue("SEBAK_TLS_KEY", "sebak.key")

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
	var err error

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options]")

		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	// flags
	flags.StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	flags.StringVar(&flagLogOutput, "log-output", flagLogOutput, "set log output file")
	flags.StringVar(&flagBind, "bind", flagBind, "address to listen on ('host:port' or ':port')")
	flags.StringVar(&flagTLSCertFile, "tls-cert", flagTLSCertFile, "tls certificate file")
	flags.StringVar(&flagTLSKeyFile, "tls-Key", flagTLSKeyFile, "tls Keyificate file")

	flags.Parse(os.Args[1:])

	if _, err := os.Stat(flagTLSCertFile); os.IsNotExist(err) {
		printFlagsError("-tls-cert", err)
	}
	if _, err := os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
		printFlagsError("-tls-key", err)
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

	log.Info("Starting Sebak")

	// print flags
	parsedFlags := []interface{}{}
	parsedFlags = append(parsedFlags, "log-level", flagLogLevel)
	parsedFlags = append(parsedFlags, "log-output", flagLogOutput)
	parsedFlags = append(parsedFlags, "bind", flagBind)
	parsedFlags = append(parsedFlags, "tls-cert", flagTLSCertFile)
	parsedFlags = append(parsedFlags, "tls-key", flagTLSKeyFile)

	log.Debug("parsed flags:", parsedFlags...)

	// NOTE instead of set `http2.VerboseLogs`, just use
	// `GODEBUG="http2debug=2"`.
	/*
		if logLevel == logging.LvlDebug {
			http2.VerboseLogs = true
		}
	*/
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := network.HTTP2TransportConfig{
		Addr:              flagBind,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       5 * time.Second,
		TLSCertFile:       flagTLSCertFile,
		TLSKeyFile:        flagTLSKeyFile,
	}
	transport := network.NewHTTP2Transport(config)

	go func() {
		log.Debug("transport started", "transport", transport, "endpoint", transport.Endpoint())

		if err := transport.Start(); err != nil {
			log.Crit("transport error", "error", err)

			os.Exit(1)
		}
	}()

	transport.AddHandler("/", TestFindme)
	transport.AddHandler("/event", TestEvent)
	transport.AddHandler("/message", TestMessage)

	transport.Ready()

	for i := range transport.ReceiveMessage() {
		fmt.Println("KKKKKKKK", i.Type, string(i.Data))
	}
	select {}

	os.Exit(0)
}
