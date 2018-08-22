package block

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
)

// BlockAccount is account model in block. the storage should support,
//  * find by `Address`:
// 	- key: `Address`: value: `ID` of BlockAccount
//  * get list by created order:
//
// models
//  * 'address'
// 	- 'ba-address-<BlockAccount.Address>': `BlockAccount`
//  * 'created'
// 	- 'ba-created-<sequential uuid1>': `BlockAccouna.Address`

const BlockAccountPrefixAddress string = "ba-address-"
const BlockAccountPrefixCreated string = "ba-created-"
const BlockAccountCheckpointPrefix string = "bac-ac-"
const BlockAccountCheckpointByAddressPrefix string = "bac-aa-"

type BlockAccount struct {
	Address    string
	Balance    string
	Checkpoint string
	CodeHash   []byte
	RootHash   sebakcommon.Hash
}

func NewBlockAccount(address string, balance sebakcommon.Amount, checkpoint string) *BlockAccount {
	return &BlockAccount{
		Address:    address,
		Balance:    balance.String(),
		Checkpoint: checkpoint,
	}
}

func (b *BlockAccount) String() string {
	return string(sebakcommon.MustJSONMarshal(b))
}

func (b *BlockAccount) Save(st *sebakstorage.LevelDBBackend) (err error) {
	key := GetBlockAccountKey(b.Address)

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
		createdKey := GetBlockAccountCreatedKey(sebakcommon.GetUniqueIDFromUUID())
		err = st.New(createdKey, b.Address)
	}
	if err == nil {
		event := "saved"
		event += " " + fmt.Sprintf("address-%s", b.Address)
		observer.BlockAccountObserver.Trigger(event, b)
	}

	bac := BlockAccountCheckpoint{
		Checkpoint: b.Checkpoint,
		Address:    b.Address,
		Balance:    b.Balance,
	}
	err = bac.Save(st)

	return
}

func (b *BlockAccount) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(b)
	return
}
func (b *BlockAccount) Deserialize(encoded []byte) (err error) {
	return sebakcommon.DecodeJSONValue(encoded, b)
}

func GetBlockAccountKey(address string) string {
	return fmt.Sprintf("%s%s", BlockAccountPrefixAddress, address)
}

func GetBlockAccountCreatedKey(created string) string {
	return fmt.Sprintf("%s%s", BlockAccountPrefixCreated, created)
}

func ExistBlockAccount(st *sebakstorage.LevelDBBackend, address string) (exists bool, err error) {
	return st.Has(GetBlockAccountKey(address))
}

func GetBlockAccount(st *sebakstorage.LevelDBBackend, address string) (b *BlockAccount, err error) {
	if err = st.Get(GetBlockAccountKey(address), &b); err != nil {
		return
	}

	return
}

func GetBlockAccountAddressesByCreated(st *sebakstorage.LevelDBBackend, reverse bool) (func() (string, bool), func()) {
	iterFunc, closeFunc := st.GetIterator(BlockAccountPrefixCreated, reverse)

	return (func() (string, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return "", false
			}

			var address string
			json.Unmarshal(item.Value, &address)
			return address, hasNext
		}), (func() {
			closeFunc()
		})
}

func GetBlockAccountsByCreated(st *sebakstorage.LevelDBBackend, reverse bool) (func() (*BlockAccount, bool), func()) {
	iterFunc, closeFunc := GetBlockAccountAddressesByCreated(st, reverse)

	return (func() (*BlockAccount, bool) {
			address, hasNext := iterFunc()
			if !hasNext {
				return nil, false
			}

			ba, err := GetBlockAccount(st, address)

			// TODO if err != nil, stopping iteration is right? how about just
			// ignoring the missing one?
			if err != nil {
				return nil, false
			}
			return ba, hasNext
		}), (func() {
			closeFunc()
		})
}

func (b *BlockAccount) GetBalance() sebakcommon.Amount {
	return sebakcommon.MustAmountFromString(b.Balance)
}

// Add fund to an account
//
// If the amount would make the account overflow over the full supply of coin,
// an `error` is returned.
func (b *BlockAccount) Deposit(fund sebakcommon.Amount, checkpoint string) error {
	if val, err := b.GetBalance().Add(fund); err != nil {
		return err
	} else {
		b.Balance = val.String()
		b.Checkpoint = checkpoint
	}
	return nil
}

// Remove fund from an account
//
// If the amount would make the account go negative, an `error` is returned.
func (b *BlockAccount) Withdraw(fund sebakcommon.Amount, checkpoint string) error {
	if val, err := b.GetBalance().Sub(fund); err != nil {
		return err
	} else {
		b.Balance = val.String()
		b.Checkpoint = checkpoint
	}
	return nil
}

// BlockAccountCheckpoint is the one-and-one model of account and checkpoint in
// block. the storage should support,
//  * find by `Address`:
// 	- key: "`Address`-`Checkpoint`": value: `ID` of BlockAccountCheckpoint
//  * get list by created order:
//
// models
//  * 'address' and 'checkpoint'
// 	- 'bac-<BlockAccountCheckpoint.Address>-<BlockAccountCheckpoint.Checkpoint>': `BlockAccountCheckpoint`
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
