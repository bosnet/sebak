package sebak

import (
	"fmt"
	"sort"

	sebakcommon "boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/storage"
	sebakstorage "boscoin.io/sebak/lib/storage"

	"github.com/btcsuite/btcutil/base58"
)

type StateClone struct {
	db    *sebakstorage.LevelDBBackend
	store *StateStore

	accounts map[string]*BlockAccount
	objects  map[string]*StateObject
}

func NewStateClone(store *StateStore) *StateClone {

	s := &StateClone{
		db:       store.DBBackend(),
		store:    store,
		accounts: make(map[string]*BlockAccount),
		objects:  make(map[string]*StateObject),
	}

	return s
}

// Commit current changed states to db
func (s *StateClone) Commit() error {

	ts, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	for _, a := range s.accounts {
		if err := a.Save(ts); err != nil {
			return err
		}
	}

	for key, obj := range s.objects {
		value := obj.Value

		switch obj.State {
		case StateObjectChanged:
			isUpdated := false

			if ok, err := ts.Has(key); err != nil {
				return err
			} else if ok {
				isUpdated = true
			}

			if isUpdated == true {
				if err := ts.Set(key, value); err != nil {
					return err
				}
			} else {
				if err := ts.New(key, value); err != nil {
					return err
				}
			}
		case StateObjectDeleted:
			if err := ts.Remove(key); err != nil {
				return err
			}
		}
	}

	if err := ts.Commit(); err != nil {
		if derr := ts.Discard(); derr != nil {
			return fmt.Errorf("err: commit:%v discard: %v", err, derr)
		}
		return err
	}

	return nil
}

func (s *StateClone) MakeHash() ([]byte, error) {

	objkeys := make([]string, 0, len(s.objects))
	accountkeys := make([]string, 0, len(s.accounts))

	for k := range s.objects {
		objkeys = append(objkeys, k)
	}
	for k := range s.accounts {
		accountkeys = append(accountkeys, k)
	}

	sort.Strings(objkeys)
	sort.Strings(accountkeys)

	hashes := make([][]byte, 0, len(s.objects)+len(s.accounts))

	for _, k := range objkeys {
		h, err := sebakcommon.MakeObjectHash(s.objects[k])
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}
	for _, k := range accountkeys {
		h, err := sebakcommon.MakeObjectHash(s.accounts[k])
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, h)
	}

	h, err := sebakcommon.MakeObjectHash(hashes)
	if err != nil {
		return nil, err
	}
	return h, nil

}

func (s *StateClone) MakeHashString() (string, error) {
	h, err := s.MakeHash()
	if err != nil {
		return "", err
	}
	return base58.Encode(h), nil
}

func (s *StateClone) GetAccount(addr string) (*BlockAccount, error) {
	if a, ok := s.accounts[addr]; ok {
		return a, nil
	}

	a, err := s.store.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *StateClone) AccountWithdraw(addr string, fund sebakcommon.Amount, checkpoint string) error {
	a, err := s.GetAccount(addr)
	if err != nil {
		return err
	}

	if err := a.Withdraw(fund, checkpoint); err != nil {
		return err
	}

	if _, ok := s.accounts[addr]; !ok {
		s.accounts[addr] = a
	}

	return nil
}

func (s *StateClone) PutDeployCode(deployCode *payload.DeployCode) error {
	key := getContractCodeKey(deployCode.ContractAddress)

	s.objects[key] = &StateObject{
		Value: deployCode,
		State: StateObjectChanged,
	}

	return nil
}

func (s *StateClone) GetDeployCode(addr string) (*payload.DeployCode, error) {
	key := getContractCodeKey(addr)
	if s.isStateDeleted(key) {
		return nil, nil
	}

	if obj, ok := s.objects[key]; ok {
		deployCode := obj.Value.(*payload.DeployCode)
		return deployCode, nil
	}

	return s.store.GetDeployCode(addr)
}

func (s *StateClone) GetStorageItem(addr, key string) (*storage.StorageItem, error) {
	itemKey := getContractStorageItemKey(addr, key)
	if s.isStateDeleted(itemKey) {
		return nil, nil
	}

	if obj, ok := s.objects[itemKey]; ok {
		item := obj.Value.(*storage.StorageItem)
		return item, nil
	}

	item, err := s.store.GetStorageItem(addr, key)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *StateClone) PutStorageItem(addr, key string, item *storage.StorageItem) error {
	itemKey := getContractStorageItemKey(addr, key)

	s.objects[itemKey] = &StateObject{
		Value: item,
		State: StateObjectChanged,
	}
	return nil
}

func (s *StateClone) DeleteStorageItem(addr, key string) error {
	itemKey := getContractStorageItemKey(addr, key)
	if v, ok := s.objects[itemKey]; ok {
		v.Value = nil
		v.State = StateObjectDeleted
	} else {
		s.objects[itemKey] = &StateObject{
			State: StateObjectDeleted,
		}
	}
	return nil
}

func (s *StateClone) isStateDeleted(key string) bool {
	if v, ok := s.objects[key]; ok {
		if v.State == StateObjectDeleted {
			return true
		}
	}
	return false
}
