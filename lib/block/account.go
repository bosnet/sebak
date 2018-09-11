package block

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
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
const BlockAccountSequenceIDPrefix string = "bac-ac-"
const BlockAccountSequenceIDByAddressPrefix string = "bac-aa-"

type BlockAccount struct {
	Address    string
	Balance    common.Amount
	SequenceID uint64
	CodeHash   []byte
	RootHash   common.Hash
}

func NewBlockAccount(address string, balance common.Amount) *BlockAccount {
	return &BlockAccount{
		Address:    address,
		Balance:    balance,
		SequenceID: 0,
	}
}

func (b *BlockAccount) String() string {
	return string(common.MustJSONMarshal(b))
}

func (b *BlockAccount) Save(st *storage.LevelDBBackend) (err error) {
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
		createdKey := GetBlockAccountCreatedKey(common.GetUniqueIDFromUUID())
		err = st.New(createdKey, b.Address)
	}
	if err == nil {
		event := "saved"
		event += " " + fmt.Sprintf("address-%s", b.Address)
		observer.BlockAccountObserver.Trigger(event, b)
	}

	bac := BlockAccountSequenceID{
		SequenceID: b.SequenceID,
		Address:    b.Address,
		Balance:    b.GetBalance(),
	}
	err = bac.Save(st)

	return
}

func (b *BlockAccount) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(b)
	return
}
func (b *BlockAccount) Deserialize(encoded []byte) (err error) {
	return common.DecodeJSONValue(encoded, b)
}

func GetBlockAccountKey(address string) string {
	return fmt.Sprintf("%s%s", BlockAccountPrefixAddress, address)
}

func GetBlockAccountCreatedKey(created string) string {
	return fmt.Sprintf("%s%s", BlockAccountPrefixCreated, created)
}

func ExistBlockAccount(st *storage.LevelDBBackend, address string) (exists bool, err error) {
	return st.Has(GetBlockAccountKey(address))
}

func GetBlockAccount(st *storage.LevelDBBackend, address string) (b *BlockAccount, err error) {
	if err = st.Get(GetBlockAccountKey(address), &b); err != nil {
		return
	}

	return
}

func GetBlockAccountAddressesByCreated(st *storage.LevelDBBackend, reverse bool) (func() (string, bool), func()) {
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

func GetBlockAccountsByCreated(st *storage.LevelDBBackend, reverse bool) (func() (*BlockAccount, bool), func()) {
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

func (b *BlockAccount) GetBalance() common.Amount {
	return b.Balance
}

// Add fund to an account
//
// If the amount would make the account overflow over the full supply of coin,
// an `error` is returned.
func (b *BlockAccount) Deposit(fund common.Amount) error {
	if val, err := b.GetBalance().Add(fund); err != nil {
		return err
	} else {
		b.Balance = val
	}
	return nil
}

// Remove fund from an account
//
// If the amount would make the account go negative, an `error` is returned.
func (b *BlockAccount) Withdraw(fund common.Amount, sequenceID uint64) error {
	if val, err := b.GetBalance().Sub(fund); err != nil {
		return err
	} else {
		b.Balance = val
		b.SequenceID = sequenceID
	}
	return nil
}

// BlockAccountSequenceID is the one-and-one model of account and sequenceID in
// block. the storage should support,
//  * find by `Address`:
// 	- key: "`Address`-`SequenceID`": value: `ID` of BlockAccountSequenceID
//  * get list by created order:
//
// models
//  * 'address' and 'sequenceID'
// 	- 'bac-<BlockAccountSequenceID.Address>-<BlockAccountSequenceID.SequenceID>': `BlockAccountSequenceID`
type BlockAccountSequenceID struct {
	SequenceID uint64
	Address    string
	Balance    common.Amount
}

func GetBlockAccountSequenceIDKey(address string, sequenceID uint64) string {
	return fmt.Sprintf("%s%s-%v", BlockAccountSequenceIDPrefix, address, sequenceID)
}

func GetBlockAccountSequenceIDByAddressKey(address string) string {
	return fmt.Sprintf("%s%s-%s", BlockAccountSequenceIDByAddressPrefix, address, common.GetUniqueIDFromUUID())
}

func GetBlockAccountSequenceIDByAddressKeyPrefix(address string) string {
	return fmt.Sprintf("%s%s-", BlockAccountSequenceIDByAddressPrefix, address)
}

func (b *BlockAccountSequenceID) String() string {
	return string(common.MustJSONMarshal(b))
}

func (b *BlockAccountSequenceID) Save(st *storage.LevelDBBackend) (err error) {
	key := GetBlockAccountSequenceIDKey(b.Address, b.SequenceID)

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
		keyByAddress := GetBlockAccountSequenceIDByAddressKey(b.Address)
		err = st.New(keyByAddress, key)
	}

	return
}

func GetBlockAccountSequenceID(st *storage.LevelDBBackend, address string, sequenceID uint64) (b BlockAccountSequenceID, err error) {
	if err = st.Get(GetBlockAccountSequenceIDKey(address, sequenceID), &b); err != nil {
		return
	}

	return
}

func GetBlockAccountSequenceIDByAddress(st *storage.LevelDBBackend, address string, reverse bool) (func() (BlockAccountSequenceID, bool), func()) {
	prefix := GetBlockAccountSequenceIDByAddressKeyPrefix(address)
	iterFunc, closeFunc := st.GetIterator(prefix, reverse)

	return (func() (BlockAccountSequenceID, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockAccountSequenceID{}, false
			}

			var key string
			json.Unmarshal(item.Value, &key)

			var bac BlockAccountSequenceID
			if err := st.Get(key, &bac); err != nil {
				return BlockAccountSequenceID{}, false
			}
			return bac, hasNext
		}), (func() {
			closeFunc()
		})
}
