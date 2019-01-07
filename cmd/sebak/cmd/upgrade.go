package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/util"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

var (
	flagStorage string
)

func init() {
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the DB from a previously existing version (or do nothing)",
		Run:   runUpgrade,
	}

	flagStorage = common.GetENVValue("SEBAK_STORAGE", cmdcommon.GetDefaultStoragePath(upgradeCmd))
	upgradeCmd.Flags().StringVar(&flagStorage, "storage", flagStorage, "storage uri")

	rootCmd.AddCommand(upgradeCmd)
}

// Entry point of the command
func runUpgrade(c *cobra.Command, args []string) {
	// Open the DB
	var st *storage.LevelDBBackend
	if storageConfig, err := storage.NewConfigFromString(flagStorage); err != nil {
		cmdcommon.PrintFlagsError(c, "--storage", err)
	} else if st, err = storage.NewStorage(storageConfig); err != nil {
		cmdcommon.PrintFlagsError(c, "--storage", err)
	}
	if err := apply20190107UpgradeBlockOperation(st); err != nil {
		fmt.Println("Upgrade 2019-01-07 failed; nothing was changed; Error: ", err)
	} else {
		fmt.Println("Triggering compaction of the DB...")
		st.DB.CompactRange(util.Range{})
	}
}

// 2019-01-07: BlockOperation is binary serialized instead of JSON
func apply20190107UpgradeBlockOperation(st *storage.LevelDBBackend) error {
	// Start to iterate over all operations, and if we can deserialize the first
	// one as JSON, keep going
	iter, closer := st.GetIterator(common.BlockOperationPrefixHash, nil)
	defer closer()
	item, hasNext := iter()
	if item.N == 0 {
		return fmt.Errorf("<upgrade> will not run on an empty storage")
	}
	var bo block.BlockOperation
	// If deserialization succeed, the data is not binary
	if err := json.Unmarshal(item.Value, &bo); err != nil {
		fmt.Println("Upgrade 2019-01-07: Already applied")
		return nil
	}

	fmt.Println("Applying upgrade 2019-01-07: Binary serialization for BlockOperation")
	batch, err := st.OpenBatch()
	if err != nil {
		return err
	}
	// Write the (already deserialized) first value
	if err = batch.Set(string(item.Key), bo); err != nil {
		batch.Discard()
		return err
	}
	count := uint64(1)
	// And everything else
	for ; hasNext; item, hasNext = iter() {
		if err = json.Unmarshal(item.Value, &bo); err != nil {
			batch.Discard()
			return err
		}
		if err = batch.Set(string(item.Key), bo); err != nil {
			batch.Discard()
			return err
		}
		count = item.N
	}
	if err := batch.Commit(); err != nil {
		fmt.Println("Failed to commit a batch of ", count, " BlockOperations")
		batch.Discard()
		return err
	}
	fmt.Println("2019-01-07 applied; ", count, " items affected")
	return nil
}

// This function is called by `run` to ensure that the node DB is up to date
func needsDBUpgrade(st *storage.LevelDBBackend) bool {
	iter, closer := st.GetIterator(common.BlockOperationPrefixHash, nil)
	defer closer()
	item, _ := iter()
	if item.N == 0 {
		return false
	}
	var bo block.BlockOperation
	// If deserialization succeed, the data is not binary
	if err := json.Unmarshal(item.Value, &bo); err == nil {
		return true
	}
	return false
}
