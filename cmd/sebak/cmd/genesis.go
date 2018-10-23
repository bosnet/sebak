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
		Use:   "genesis <public key of genesis account> <public key of common account>",
		Short: "initialize new network",
		Args:  cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			flagName, err := makeGenesisBlock(args[0], args[1], flagNetworkID, flagBalance, flagStorageConfigString, log)
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

// makeGenesisBlock creates a genesis block using the provided parameter
//
// This function is separate, and public, to allow it to be used from other modules
// (at the moment, only `run`) so it can provide the same behavior (defaults, error messages).
//
// Params:
//   genesisAddress  = public address of the account owning the genesis block
//   commonAddress   = public address of the common account
//   networkID = `--network-id` argument, used for signing
//   balanceStr = Amount of coins to put in the genesis account
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
func makeGenesisBlock(genesisAddress, commonAddress, networkID, balanceStr, storageUri string, log logging.Logger) (string, error) {
	var balance common.Amount
	var err error
	var genesisKP keypair.KP
	var commonKP keypair.KP

	if genesisKP, err = keypair.Parse(genesisAddress); err != nil {
		return "<public key of genesis account>", err
	}

	if commonKP, err = keypair.Parse(commonAddress); err != nil {
		return "<public key of common account>", err
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
	defer st.Close()

	// check account does not exists
	if _, err = block.GetBlockAccount(st, genesisKP.Address()); err == nil {
		return "<public key>", errors.New("account is already created")
	}

	genesisAccount := block.NewBlockAccount(genesisKP.Address(), balance)
	if err := genesisAccount.Save(st); err != nil {
		return "<public key>", fmt.Errorf("failed to create genesis account: %v", err)
	}

	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)
	if err := commonAccount.Save(st); err != nil {
		return "<public key>", fmt.Errorf("failed to create common account: %v", err)
	}

	b, err := block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, []byte(flagNetworkID))
	if err != nil {
		return "<public key>", fmt.Errorf("failed to create genesis block: %v", err)
	}

	log.Info("genesis block created",
		"height", b.Height,
		"round", b.Round.Number,
		"timestamp", b.Timestamp,
		"total-txs", b.TotalTxs,
		"proposer", b.Proposer,
	)

	return "", nil
}
