package cmd

import (
	"os"

	logging "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"

	"boscoin.io/sebak/lib/network"

	"boscoin.io/sebak/cmd/sebak/common"
)

var (
	tlsCmd *cobra.Command
)

func init() {
	tlsCmd = &cobra.Command{
		Use:   "tls",
		Short: "Generate simple tls cert and key file",
		Run: func(c *cobra.Command, args []string) {
			generate()
			return
		},
	}

	tlsCmd.Flags().StringVar(&flagTLSCertFile, "cert-name", flagTLSCertFile, "tls certificate file name")
	tlsCmd.Flags().StringVar(&flagTLSKeyFile, "key-name", flagTLSKeyFile, "tls key file name")

	rootCmd.AddCommand(tlsCmd)
}

func generate() {
	var err error

	sebaknetwork.GenerateKey(".", flagTLSCertFile, flagTLSKeyFile)

	if _, err = os.Stat(flagTLSCertFile); os.IsNotExist(err) {
		common.PrintFlagsError(tlsCmd, "cert", err)
	}
	if _, err = os.Stat(flagTLSKeyFile); os.IsNotExist(err) {
		common.PrintFlagsError(tlsCmd, "key", err)
	}

	log = logging.New("module", "tls")

	log.Info("Generate tls cert and key files", "cert", flagTLSCertFile, "key", flagTLSKeyFile)

}
