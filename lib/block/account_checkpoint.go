package block

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"encoding/json"
	"fmt"
)

// BlockAccountCheckpoint is the one-and-one model of account and checkpoint in
// block. the storage should support,
//  * find by `Address`:
// 	- key: "`Address`-`Checkpoint`": value: `ID` of BlockAccountCheckpoint
//  * get list by created order:
//
// models
//  * 'address' and 'checkpoint'
// 	- 'bac-<BlockAccountCheckpoint.AdAdress>-<BlockAccountCheckpoint.Checkpoint>': `BlockAccountCheckpoint`

const BlockAccountCheckpointPrefix string = "bac-ac-"
const BlockAccountCheckpointByAddressPrefix string = "bac-aa-"

type BlockAccountCheckpoint struct {
	Checkpoint string
	Address    string
	Balance    string
}

func GetBlockAccountCheckpointKey(address, checkpoint string) string {
	return fmt.Sprintf("%s%s-%s", BlockAccountCheckpointPrefix, address, checkpoint)
}

func GetBlockAccountCheckpointByAddressKey(address string) string {
	return fmt.Sprintf("%s%s-%s", BlockAccountCheckpointByAddressPrefix, address, sebakcommon.GetUniqueIDFromUUID())
}

func GetBlockAccountCheckpointByAddressKeyPrefix(address string) string {
	return fmt.Sprintf("%s%s-", BlockAccountCheckpointByAddressPrefix, address)
}

func (b *BlockAccountCheckpoint) String() string {
	return string(sebakcommon.MustJSONMarshal(b))
}

func (b *BlockAccountCheckpoint) Save(st *sebakstorage.LevelDBBackend) (err error) {
	key := GetBlockAccountCheckpointKey(b.Address, b.Checkpoint)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	}

	if exists {
		err = st.Set(key, b)
	} else {
		// TODO consider to use, [`Transaction`](https://godoc.org/github.com/syndtr/goleveldb/leveldb#DB.OpenTransaction)
		err = st.New(key, b)
	}

	if !exists {
		keyByAddress := GetBlockAccountCheckpointByAddressKey(b.Address)
		err = st.New(keyByAddress, key)
	}

	return
}

func GetBlockAccountCheckpoint(st *sebakstorage.LevelDBBackend, address, checkpoint string) (b BlockAccountCheckpoint, err error) {
	if err = st.Get(GetBlockAccountCheckpointKey(address, checkpoint), &b); err != nil {
		return
	}

	return
}

func GetBlockAccountCheckpointByAddress(st *sebakstorage.LevelDBBackend, address string, reverse bool) (func() (BlockAccountCheckpoint, bool), func()) {
	prefix := GetBlockAccountCheckpointByAddressKeyPrefix(address)
	iterFunc, closeFunc := st.GetIterator(prefix, reverse)

	return (func() (BlockAccountCheckpoint, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockAccountCheckpoint{}, false
			}

			var key string
			json.Unmarshal(item.Value, &key)

			var bac BlockAccountCheckpoint
			if err := st.Get(key, &bac); err != nil {
				return BlockAccountCheckpoint{}, false
			}
			return bac, hasNext
		}), (func() {
			closeFunc()
		})
}
