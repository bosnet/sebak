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

type BlockAccount struct {
	Address    string        `json:"address"`
	Balance    common.Amount `json:"balance"`
	SequenceID uint64        `json:"sequence_id"`
	// An address, or "" if the account isn't frozen
	Linked   string      `json:"linked"`
	CodeHash []byte      `json:"code_hash"`
	RootHash common.Hash `json:"root_hash"`
}

func NewBlockAccount(address string, balance common.Amount) *BlockAccount {
	return NewBlockAccountLinked(address, balance, "")
}

func NewBlockAccountLinked(address string, balance common.Amount, linked string) *BlockAccount {
	return &BlockAccount{
		Address:    address,
		Balance:    balance,
		SequenceID: 0,
		Linked:     linked,
	}
}

func (b *BlockAccount) IsFrozen() bool {
	return b.Linked != ""
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
		err = st.New(key, b)
		createdKey := GetBlockAccountCreatedKey(common.GetUniqueIDFromUUID())
		err = st.New(createdKey, b.Address)
	}
	if err != nil {
		return err
	}

	if err == nil {
		event := "saved"
		event += " " + fmt.Sprintf("address-%s", b.Address)
		observer.BlockAccountObserver.Trigger(event, b)
	}

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
	return fmt.Sprintf("%s%s", common.BlockAccountPrefixAddress, address)
}

func GetBlockAccountCreatedKey(created string) string {
	return fmt.Sprintf("%s%s", common.BlockAccountPrefixCreated, created)
}

func ExistsBlockAccount(st *storage.LevelDBBackend, address string) (exists bool, err error) {
	return st.Has(GetBlockAccountKey(address))
}

func GetBlockAccount(st *storage.LevelDBBackend, address string) (b *BlockAccount, err error) {
	if err = st.Get(GetBlockAccountKey(address), &b); err != nil {
		return
	}

	return
}

func GetBlockAccountAddressesByCreated(st *storage.LevelDBBackend, options storage.ListOptions) (func() (string, bool, []byte), func()) {
	iterFunc, closeFunc := st.GetIterator(common.BlockAccountPrefixCreated, options)

	return (func() (string, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return "", false, []byte{}
			}

			var address string
			json.Unmarshal(item.Value, &address)
			return address, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlockAccountsByCreated(st *storage.LevelDBBackend, options storage.ListOptions) (func() (*BlockAccount, bool, []byte), func()) {
	iterFunc, closeFunc := GetBlockAccountAddressesByCreated(st, options)

	return (func() (*BlockAccount, bool, []byte) {
			address, hasNext, cursor := iterFunc()
			if !hasNext {
				return nil, false, cursor
			}

			ba, err := GetBlockAccount(st, address)

			// TODO if err != nil, stopping iteration is right? how about just
			// ignoring the missing one?
			if err != nil {
				return nil, false, cursor
			}
			return ba, hasNext, cursor
		}), (func() {
			closeFunc()
		})
}

func LoadBlockAccountsInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (*BlockAccount, bool, []byte),
	func(),
) {

	return (func() (*BlockAccount, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return &BlockAccount{}, false, item.Key
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			ba, err := GetBlockAccount(st, hash)
			if err != nil {
				return &BlockAccount{}, false, item.Key
			}

			return ba, hasNext, item.Key
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
func (b *BlockAccount) Withdraw(fund common.Amount) error {
	if val, err := b.GetBalance().Sub(fund); err != nil {
		return err
	} else {
		b.Balance = val
		b.SequenceID += 1
	}
	return nil
}
