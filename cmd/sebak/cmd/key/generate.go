package key

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/cmd/sebak/common"
)

var (
	GenerateCmd *cobra.Command

	flagInput     string
	flagShort     bool
	flagPublicKey bool
)

func init() {
	GenerateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Generate keypair",
		Run: func(c *cobra.Command, args []string) {
			if len(args) > 0 {
				flagInput = strings.TrimSpace(strings.Join(args, " "))
			} else if flagPublicKey && len(flagInput) < 1 {
				common.PrintFlagsError(c, "--publicKey", errors.New("--publicKey needs <public key>"))
			}

			kp, err := generateKP()
			if err != nil {
				common.PrintFlagsError(c, "<input>", fmt.Errorf("failed to parse public key: %v", err))
			}

			if flagShort {
				fmt.Fprintf(os.Stdout, "%s %s\n", kp.Seed(), kp.Address())

				os.Exit(0)
			}

			t := template.Must(template.New("").Parse(`       Secret Seed: {{ .seed }}
    Public Address: {{ .address }}{{ if .hasNetworkPassphrase }}
Network Passphrase: "{{ .networkPassphrase}}"{{ end }}
`))
			t.Execute(os.Stdout, map[string]interface{}{
				"address":              kp.Address(),
				"seed":                 kp.Seed(),
				"networkPassphrase":    flagInput,
				"hasNetworkPassphrase": !flagPublicKey && len(flagInput) > 0,
			})
		},
	}

	GenerateCmd.Flags().BoolVar(&flagShort, "short", false, "short format, \"<secret seed> <public address>\"")
	GenerateCmd.Flags().BoolVar(&flagPublicKey, "publicKey", false, "parse public key")
	//GenerateCmd.Flags().StringVar(&flagNetworkPassphrase, "short", false, "short format, \"<secret seed> <public address>\"")
}

func generateKP() (full *keypair.Full, err error) {
	if len(flagInput) < 1 {
		full, _ = keypair.Random()
		return
	}

	if flagPublicKey {
		var kp keypair.KP
		kp, err = keypair.Parse(flagInput)
		if err != nil {
			return
		}

		full = kp.(*keypair.Full)
	}

	full = keypair.Master(flagInput).(*keypair.Full)
	return
}
