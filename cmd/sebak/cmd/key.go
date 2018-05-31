package cmd

import (
	"github.com/spf13/cobra"

	"github.com/spikeekips/sebak/cmd/sebak/cmd/key"
)

var (
	keyCmd *cobra.Command
)

func init() {
	keyCmd = &cobra.Command{
		Use:   "key",
		Short: "Keypair management",
		Run: func(c *cobra.Command, args []string) {
			if len(args) < 1 {
				c.Usage()
			}
		},
	}

	keyCmd.AddCommand(key.GenerateCmd)
	rootCmd.AddCommand(keyCmd)
}
