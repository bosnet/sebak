package sebak

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/storage"
)

/*
BlockAccount is account model in block. the storage should support,
 * find by `Address`:
	- key: `Address`: value: `ID` of BlockAccount
 * get list by created order:

models
 * 'address'
	- 'ba-address-<BlockAccount.Address>': `BlockAccount`
 * 'created'
	- 'ba-created-<sequential uuid1>': `BlockAccouna.Address`
*/

const BlockAccountPrefixAddress string = "ba-address-"
const BlockAccountPrefixCreated string = "ba-created-"

type BlockAccount struct {
	Address    string
	Balance    string
	Checkpoint string
}

func NewBlockAccount(address, balance, checkpoint string) *BlockAccount {
	return &BlockAccount{
		Address:    address,
		Balance:    balance,
		Checkpoint: checkpoint,
	}
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
		// TODO consider to use
		// [`Transaction`](https://godoc.org/github.com/syndtr/goleveldb/leveldb#DB.OpenTransaction)
		err = st.New(key, b)
		createdKey := GetBlockAccountCreatedKey(sebakcommon.GetUniqueIDFromUUID())
		err = st.New(createdKey, b.Address)
	}

	return
}

func (b *BlockAccount) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(b)
	return
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

func (b *BlockAccount) GetBalance() int64 {
	n, _ := strconv.ParseInt(b.Balance, 10, 64)
	return n
}

func (b *BlockAccount) setBalance(n int64) (err error) {
	if n < 0 {
		err = errors.New("wrong balance value")
		return
	}

	b.Balance = strconv.FormatInt(n, 10)

	return
}

func (b *BlockAccount) EnsureUpdate(balance int64, checkpoint string, expectedBalance int64) (err error) {
	n := b.GetBalance() + balance
	if n != expectedBalance {
		err = errors.New("unexpected balance")
		return
	}

	b.setBalance(n)
	b.Checkpoint = checkpoint

	return
}
