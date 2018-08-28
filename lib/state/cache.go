package state

import (
	"fmt"
	"sort"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/storage"
)

type Cache struct {
	reader   Reader
	accounts map[string]*accountInfo
}

type accountInfo struct {
	account *block.BlockAccount
	items   map[string]*storage.Item
	updated bool
	removed bool //TODO Remove Account?
}

var _ ReadWriter = (*Cache)(nil)

func NewCache(reader Reader) *Cache {
	cache := &Cache{
		reader:   reader,
		accounts: make(map[string]*accountInfo),
	}

	return cache
}

func (c *Cache) Flush(w Writer) error {
	if err := c.Sync(w); err != nil {
		return err
	}
	if err := c.Reset(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Sync(w Writer) error {
	var addressList []string
	for address := range c.accounts {
		addressList = append(addressList, address)
	}
	sort.Strings(addressList)

	for _, address := range addressList {
		aInfo := c.accounts[address]
		if aInfo.updated {
			var keys []string
			for key := range aInfo.items {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				item := aInfo.items[key]
				if err := w.SetStorageItem(address, key, item); err != nil {
					return err
				}
			}

			if err := w.SetAccount(aInfo.account); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cache) Reset() error {
	c.accounts = make(map[string]*accountInfo)
	return nil
}

func (c *Cache) GetAccount(address string) (*block.BlockAccount, error) {
	a, err := c.get(address)
	if err != nil {
		return nil, err
	}
	if a.removed == true {
		return nil, err
	}
	return a.account, nil
}

func (c *Cache) SetAccount(account *block.BlockAccount) error {
	a, err := c.get(account.Address)
	if err != nil {
		return err
	}
	if a != nil {
		c.accounts[account.Address] = a
		return nil
	}

	a = &accountInfo{
		account: account,
		items:   map[string]*storage.Item{},
		updated: true,
	}
	c.accounts[account.Address] = a
	return nil
}

func (c *Cache) GetStorageItem(address, key string) (*storage.Item, error) {
	a, err := c.get(address)
	if err != nil {
		return nil, err
	}

	if a == nil {
		return nil, fmt.Errorf("GetStorageItem on a empty account: %s", address)
	}

	if item := a.items[key]; item != nil {
		return item, nil
	}

	item, err := c.reader.GetStorageItem(address, key)
	if err != nil {
		return nil, err
	}

	a.items[key] = item

	return item, nil
}

func (c *Cache) SetStorageItem(address, key string, item *storage.Item) error {
	a, err := c.get(address)
	if err != nil {
		return err
	}

	if a == nil {
		return fmt.Errorf("SetStorageItem on a empty account: %s", address)
	}

	a.items[key] = item
	a.updated = true

	return nil
}

func (c *Cache) get(address string) (*accountInfo, error) {
	a := c.accounts[address]
	if a == nil {
		ba, err := c.reader.GetAccount(address)
		if err != nil {
			return nil, err
		}
		if ba == nil {
			return nil, nil
		}

		a = &accountInfo{
			account: ba,
			items:   make(map[string]*storage.Item),
		}
		c.accounts[address] = a
	}
	return a, nil

}
