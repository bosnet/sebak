package cmd

import (
	"boscoin.io/sebak/cmd/sebak/cmd/wallet"

	"github.com/spf13/cobra"
)

var (
	walletCmd *cobra.Command
)

func init() {
	walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "CLI for wallet management",
		Run: func(c *cobra.Command, args []string) {
			if len(args) < 1 {
				c.Usage()
			}
		},
	}

	rootCmd.AddCommand(walletCmd)
	walletCmd.AddCommand(wallet.PaymentCmd)
	walletCmd.AddCommand(wallet.UnfreezeRequestCmd)
}
