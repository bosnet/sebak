package common

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func PrintFlagsError(cmd *cobra.Command, flagName string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid '%s'; %v\n\n", flagName, err)
	}

	cmd.Help()

	os.Exit(1)
}
