package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"boscoin.io/sebak/cmd/sebak/common"
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "sebak",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			c.Usage()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		common.PrintFlagsError(rootCmd, "", err)
	}
}

func SetArgs(s []string) {
	rootCmd.SetArgs(s)
}
