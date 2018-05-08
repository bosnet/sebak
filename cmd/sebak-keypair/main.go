package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/stellar/go/keypair"
)

var flagNetworkPassphrase string
var flagShort bool
var hasPhrase = false

func init() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.BoolVar(&flagShort, "short", false, "short format, \"<secret seed> <public address>\"")

	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] [<network passphrase>]")
		flags.PrintDefaults()
	}

	flags.Parse(os.Args[1:])
	if flags.NArg() > 0 {
		hasPhrase = true
		flagNetworkPassphrase = strings.TrimSpace(flags.Arg(0))
	}
}

func main() {
	var kp *keypair.Full
	if !hasPhrase {
		kp, _ = keypair.Random()
	} else if f, err := keypair.Parse(flagNetworkPassphrase); err == nil {
		kp = f.(*keypair.Full)
		hasPhrase = false
	} else {
		kp = keypair.Master(flagNetworkPassphrase).(*keypair.Full)
	}

	if flagShort {
		fmt.Fprintf(os.Stdout, "%s %s\n", kp.Seed(), kp.Address())
	} else {
		t := template.Must(template.New("").Parse(`       Secret Seed: {{ .seed }}
    Public Address: {{ .address }}{{ if .hasPhrase }}
Network Passphrase: '{{ .networkPassphrase}}'{{ end }}
`))
		t.Execute(os.Stdout, map[string]interface{}{
			"address":           kp.Address(),
			"seed":              kp.Seed(),
			"hasPhrase":         hasPhrase,
			"networkPassphrase": flagNetworkPassphrase,
		})
	}
}
