package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"boscoin.io/sebak/lib/version"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", version.ToDetailVersion())
	},
}
