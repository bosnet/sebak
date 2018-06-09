package key

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/owlchain/sebak/cmd/sebak/common"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"
)

var (
	GenerateCmd *cobra.Command

	flagPublicKey bool
	flagFormat    string

	encoders = common.NewEncoders(map[string]common.EncodeFn{
		"default": func(v interface{}) common.Encoder {
			return &defaultEncoder{v.(keyPair)}
		},
		"oneline": func(v interface{}) common.Encoder {
			return &onelineEncoder{v.(keyPair)}
		},
	})
)

type (
	keyPair struct {
		Seed              string  `json:"seed"`
		Address           string  `json:"address"`
		NetworkPassphrase *string `json:"network_passphrase,omitempty"`
	}
)

type defaultEncoder struct {
	kp keyPair
}

func (o *defaultEncoder) Encode(w io.Writer) error {
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
	return t.Execute(w, o.kp)
}

type onelineEncoder struct {
	kp keyPair
}

func (o *onelineEncoder) Encode(w io.Writer) error {
	fmt.Fprintf(w, "%s %s\n", o.kp.Seed, o.kp.Address)
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
				common.PrintFlagsError(c, "--publicKey", errors.New("--publicKey needs <public key>"))
			}

			kp, err := generateKP(input, flagPublicKey)

			if flagPublicKey && err != nil {
				common.PrintFlagsError(c, "<input>", fmt.Errorf("failed to parse public key: %v", err))
			} else if !flagPublicKey && len(input) > 0 {
				passphrase = &input
			}

			if encoder, ok := encoders.Get(flagFormat); ok {
				err := encoder(keyPair{
					Seed:              kp.Seed(),
					Address:           kp.Address(),
					NetworkPassphrase: passphrase,
				}).Encode(os.Stdout)

				if err != nil {
					panic(err)
				}
			} else {
				common.PrintFlagsError(c, "format", fmt.Errorf(`"%s" not recognized`, flagFormat))
			}
		},
	}

	GenerateCmd.Flags().BoolVar(&flagPublicKey, "parse", false, "parse public key")
	GenerateCmd.Flags().StringVar(&flagFormat, "format", "default", "format={default, json, oneline, prettyjson}")
}

func generateKP(seedOrNetworkPassphrase string, fromSeed bool) (full *keypair.Full, err error) {
	if len(seedOrNetworkPassphrase) == 0 {
		full, err = keypair.Random()
	} else if fromSeed {
		var kp keypair.KP

		if kp, err = keypair.Parse(seedOrNetworkPassphrase); err == nil {
			full = kp.(*keypair.Full)
		}
	} else {
		full = keypair.Master(seedOrNetworkPassphrase).(*keypair.Full)
	}

	return
}
