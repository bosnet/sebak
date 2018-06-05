package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/lib"
	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/storage"

	"github.com/owlchain/sebak/cmd/sebak/common"
)

const (
	initialBalance = "1,000,000,000,000.0000000"
)

var (
	genesisCmd  *cobra.Command
	flagBalance string = sebakcommon.GetENVValue("SEBAK_GENESIS_BALANCE", initialBalance)
)

func init() {
	var genesisCmd = &cobra.Command{
		Use:   "genesis <public key>",
		Short: "initialize new network",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			var kp keypair.KP
			var balance sebak.Amount

			if kp, err = keypair.Parse(args[0]); err != nil {
				common.PrintFlagsError(c, "<public key>", err)
			}

			if balance, err = common.ParseAmountFromString(flagBalance); err != nil {
				common.PrintFlagsError(c, "--balance", err)
			}

			if storageConfig, err = sebakstorage.NewConfigFromString(flagStorageConfigString); err != nil {
				common.PrintFlagsError(c, "--storage", err)
			}

			st, err := sebakstorage.NewStorage(storageConfig)
			if err != nil {
				common.PrintFlagsError(c, "--storage", fmt.Errorf("failed to initialize storage: %v", err))
			}

			// check account is exists
			if _, err = sebak.GetBlockAccount(st, kp.Address()); err == nil {
				common.PrintFlagsError(c, "<public key>", errors.New("account is already created"))
			}

			checkpoint := uuid.New().String()
			account := sebak.NewBlockAccount(kp.Address(), balance, checkpoint)
			account.Save(st)

			fmt.Println("successfully created genesis block")
		},
	}

	/*
	 */

	var err error
	var currentDirectory string
	if currentDirectory, err = os.Getwd(); err != nil {
		common.PrintFlagsError(genesisCmd, "--storage", err)
	}
	if currentDirectory, err = filepath.Abs(currentDirectory); err != nil {
		common.PrintFlagsError(genesisCmd, "--storage", err)
	}

	flagStorageConfigString = sebakcommon.GetENVValue("SEBAK_STORAGE", fmt.Sprintf("file://%s/db", currentDirectory))

	genesisCmd.Flags().StringVar(&flagBalance, "balance", flagBalance, "initial balance of genesis block")
	genesisCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")

	rootCmd.AddCommand(genesisCmd)
}
