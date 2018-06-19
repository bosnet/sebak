package cmd

import (
	"os"

	logging "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"

	"boscoin.io/sebak/lib/network"

	"boscoin.io/sebak/cmd/sebak/common"
)

var (
	tlsCmd            *cobra.Command
	flagTLSOutputPath = "."
)

func init() {
	tlsCmd = &cobra.Command{
		Use:   "tls",
		Short: "Generate tls certificate and key file",
		Run: func(c *cobra.Command, args []string) {
			generate()
			return
		},
	}

	tlsCmd.Flags().StringVar(&flagTLSCertFile, "cert", flagTLSCertFile, "tls certificate file name")
	tlsCmd.Flags().StringVar(&flagTLSKeyFile, "key", flagTLSKeyFile, "tls key file name")
	tlsCmd.Flags().StringVar(&flagTLSOutputPath, "output", flagTLSOutputPath, "tls output path")

	rootCmd.AddCommand(tlsCmd)
}

func generate() {
	var err error

	sebaknetwork.NewKeyGenerator(flagTLSOutputPath, flagTLSCertFile, flagTLSKeyFile)

	if _, err = os.Stat(flagTLSOutputPath); os.IsNotExist(err) {
		common.PrintFlagsError(tlsCmd, "output", err)
	}
	if _, err = os.Stat(flagTLSOutputPath + "/" + flagTLSCertFile); os.IsNotExist(err) {
		common.PrintFlagsError(tlsCmd, "cert", err)
	}
	if _, err = os.Stat(flagTLSOutputPath + "/" + flagTLSKeyFile); os.IsNotExist(err) {
		common.PrintFlagsError(tlsCmd, "key", err)
	}

	log = logging.New("module", "tls")

	log.Info("Generate tls certificate and key files", "cert", flagTLSCertFile, "key", flagTLSKeyFile, "out", flagTLSOutputPath)

}
