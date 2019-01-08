package cmd

import (
	"fmt"

	logging "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
)

const (
	initialBalance = "1,000,000,000,000.0000000"
)

var (
	flagBalance string = common.GetENVValue("SEBAK_GENESIS_BALANCE", initialBalance)
)

func init() {
	var genesisCmd = &cobra.Command{
		Use:   "genesis <public key of genesis account> <public key of common account>",
		Short: "initialize new network",
		Args:  cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			genesisKP, commonKP, balance, err := parseGenesisOption(
				args[0], args[1], flagBalance,
			)
			if err != nil {
				cmdcommon.PrintError(c, err)
			}

			flagName, err := makeGenesisBlock(genesisKP, commonKP, flagNetworkID, balance, flagStorageConfigString, log)
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
func makeGenesisBlock(genesisKP, commonKP keypair.KP, networkID string, balance common.Amount, storageUri string, log logging.Logger) (string, error) {
	var err error

	if len(networkID) == 0 {
		return "--network-id", errors.New("--network-id must be provided")
	}

	if balance == 0 {
		balance = common.MaximumBalance
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

	created, err := checkExistingAccounts(st, flagNetworkID, genesisKP.Address(), commonKP.Address(), balance)
	if err != nil {
		if created {
			return "--storage", fmt.Errorf("genesis block is already created, but: %v", err)
		} else {
			return "--storage", err
		}
	} else if created {
		if b, err := block.GetBlockByHeight(st, common.GenesisBlockHeight); err != nil {
			return "--storage", fmt.Errorf("failed to get genesis block: %v", err)
		} else {
			log.Info("genesis block already created",
				"height", b.Height,
				"round", b.Round,
				"confirmed", b.Confirmed,
				"total-txs", b.TotalTxs,
				"total-ops", b.TotalOps,
				"proposer", b.Proposer,
			)
		}

		return "", nil
	}

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
		"round", b.Round,
		"confirmed", b.Confirmed,
		"total-txs", b.TotalTxs,
		"total-ops", b.TotalOps,
		"proposer", b.Proposer,
	)

	return "", nil
}

func checkExistingAccounts(st *storage.LevelDBBackend, networkID, genesisAddress, commonAddress string, balance common.Amount) (created bool, err error) {
	// check network id
	var bt block.BlockTransaction
	if bt, err = runner.GetGenesisTransaction(st); err != nil {
		if err == errors.StorageRecordDoesNotExist {
			created = false
			err = nil
			return
		}

		return
	}

	created = true

	var genesisAccount *block.BlockAccount
	if genesisAccount, err = runner.GetGenesisAccount(st); err != nil {
		return
	}
	if genesisAccount.Address != genesisAddress {
		err = fmt.Errorf("different genesis account address")
		return
	}

	var commonAccount *block.BlockAccount
	if commonAccount, err = runner.GetCommonAccount(st); err != nil {
		return
	}
	if commonAccount.Address != commonAddress {
		err = fmt.Errorf("different common account address")
		return
	}

	var genesisBalance common.Amount
	if genesisBalance, err = runner.GetGenesisBalance(st); err != nil {
		return
	}
	if genesisBalance != balance {
		err = fmt.Errorf("different balance")
		return
	}

	var tp block.TransactionPool
	if tp, err = block.GetTransactionPool(st, bt.Hash); err != nil {
		return
	}

	tx := tp.Transaction()
	existingSignature := tx.H.Signature

	kp := keypair.Master(networkID)
	tx.Sign(kp, []byte(networkID))
	if existingSignature != tx.H.Signature {
		err = fmt.Errorf("the previous genesis block was created by different networkID")
		return
	}

	return
}
