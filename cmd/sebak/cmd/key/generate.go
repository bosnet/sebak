package key

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"io"

	"boscoin.io/sebak/cmd/sebak/common"
)

var (
	GenerateCmd *cobra.Command

	flagInput     string
	flagPublicKey bool
	flagFormat    string
)

type (
	keyPair struct {
		Seed              string  `json:"seed"`
		Address           string  `json:"address"`
		NetworkPassphrase *string `json:"network_passphrase,omitempty"`
	}
)

func defaultEncode(v interface{}, w io.Writer) error {
	t := template.Must(template.New("").Funcs(template.FuncMap{
		"valueString": func(input *string) string {
			if input == nil {
				return ""
			} else {
				return *input
			}
		},
	}).Parse(`       Secret Seed: {{ .Seed }}
    Public Address: {{ .Address }}{{ if valueString .NetworkPassphrase }}
Network Passphrase: "{{ .NetworkPassphrase|valueString }}"{{ end }}
`))
	return t.Execute(w, v)
}

func onelineEncode(v interface{}, w io.Writer) error {
	kp := v.(keyPair)
	fmt.Fprintf(w, "%s %s\n", kp.Seed, kp.Address)
	return nil
}

func init() {
	GenerateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Generate keypair",
		Run: func(c *cobra.Command, args []string) {
			var passphrase *string = nil
			input := strings.TrimSpace(strings.Join(args, " "))

			if flagPublicKey && len(input) == 0 {
				common.PrintFlagsError(c, "--parse", errors.New("--parse needs <secret seed>"))
			}

			kp, err := generateKP(input, flagPublicKey)

			if flagPublicKey && err != nil {
				common.PrintFlagsError(c, "<input>", fmt.Errorf("failed to parse secret seed: %v", err))
			} else if !flagPublicKey && len(input) > 0 {
				passphrase = &input
			}

			encoders := map[string]common.Encode{
				"json":       common.DefaultEncodes["json"],
				"prettyjson": common.DefaultEncodes["prettyjson"],
				"default":    defaultEncode,
				"oneline":    onelineEncode,
			}

			if encode, ok := encoders[flagFormat]; ok {
				err := encode(keyPair{
					Seed:              kp.Seed(),
					Address:           kp.Address(),
					NetworkPassphrase: passphrase,
				}, os.Stdout)

				os.Exit(0)
				if err != nil {
					panic(err)
				}
			} else {
				common.PrintFlagsError(c, "format", fmt.Errorf(`"%s" not recognized`, flagFormat))
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

	GenerateCmd.Flags().BoolVar(&flagPublicKey, "parse", false, "parse secret seed")
	GenerateCmd.Flags().StringVar(&flagFormat, "format", "default", "format={default, json, oneline, prettyjson}")
}

func generateKP(seedOrNetworkPassphrase string, fromSeed bool) (full *keypair.Full, err error) {
	if len(seedOrNetworkPassphrase) == 0 {
		full, err = keypair.Random()
	} else if fromSeed {
		var kp keypair.KP

		if kp, err = keypair.Parse(seedOrNetworkPassphrase); err == nil {
			if kf, ok := kp.(*keypair.Full); ok {
				full = kf
			} else {
				err = fmt.Errorf("not a secret seed")
			}
		}
	} else {
		full = keypair.Master(seedOrNetworkPassphrase).(*keypair.Full)
	}

	return
}
