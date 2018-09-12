package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	logging "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

const (
	initialBalance = "1,000,000,000,000.0000000"
)

var (
	genesisCmd  *cobra.Command
	flagBalance string = common.GetENVValue("SEBAK_GENESIS_BALANCE", initialBalance)
)

func init() {
	var genesisCmd = &cobra.Command{
		Use:   "genesis <public key>",
		Short: "initialize new network",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			flagName, err := MakeGenesisBlock(args[0], flagNetworkID, flagBalance, flagStorageConfigString, log)
			if len(flagName) != 0 || err != nil {
				cmdcommon.PrintFlagsError(c, flagName, err)
			}

			fmt.Println("successfully created genesis block")
		},
	}

	genesisCmd.Flags().StringVar(&flagBalance, "balance", flagBalance, "initial balance of genesis block")
	genesisCmd.Flags().StringVar(&flagStorageConfigString, "storage", flagStorageConfigString, "storage uri")
	genesisCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")

	rootCmd.AddCommand(genesisCmd)
}

//
// Create a genesis block using the provided parameter
//
// This function is separate, and public, to allow it to be used from other modules
// (at the moment, only `run`) so it can provide the same behavior (defaults, error messages).
//
// Params:
//   address   = public address of the account owning the genesis block
//   networkID = `--network-id` argument, used for signing
//   balanceStr = Amount of coins to put in the account
//                If not provided, `flagBalance`, which is the value set in the env
//                when called from another module, will be used
//   storageUri = URI to include storage path("file://path")
//                If not provided, a default value will be used
//
// Returns:
//   If an error happened, returns a tuple of (string, error).
//   The string argument represent the name of the flag which errored,
//   and error is the more detailed error.
//   Note that only one needs be non-`nil` for it to be considered an error.
//
func MakeGenesisBlock(addressStr, networkID, balanceStr, storageUri string, log logging.Logger) (string, error) {
	var balance common.Amount
	var err error
	var kp keypair.KP

	if kp, err = keypair.Parse(addressStr); err != nil {
		return "<address>", err
	}

	if len(networkID) == 0 {
		return "--network-id", errors.New("--network-id must be provided")
	}

	if len(balanceStr) == 0 {
		balanceStr = initialBalance
	}

	if balance, err = cmdcommon.ParseAmountFromString(balanceStr); err != nil {
		return "--balance", err
	}

	// Use the default value
	if len(storageUri) == 0 {
		// We try to get the env value first, before doing IO which could fail
		storageUri = common.GetENVValue("SEBAK_STORAGE", "")
		// No env, use the default (current directory)
		if len(storageUri) == 0 {
			if currentDirectory, err := os.Getwd(); err == nil {
				if currentDirectory, err = filepath.Abs(currentDirectory); err == nil {
					storageUri = fmt.Sprintf("file://%s/db", currentDirectory)
				}
			}
			// If any of the previous condition failed
			if len(storageUri) == 0 {
				return "--storage", err
			}
		}
	}

	var storageConfig *storage.Config
	if storageConfig, err = storage.NewConfigFromString(storageUri); err != nil {
		return "--storage", err
	}

	st, err := storage.NewStorage(storageConfig)
	if err != nil {
		return "--storage", fmt.Errorf("failed to initialize storage: %v", err)
	}

	// check account does not exists
	if _, err = block.GetBlockAccount(st, kp.Address()); err == nil {
		return "<public key>", errors.New("account is already created")
	}

	account := block.NewBlockAccount(
		kp.Address(),
		balance,
	)
	account.Save(st)

	block.MakeGenesisBlock(st, *account, log)

	st.Close()
	return "", nil
}
