package stateclone

import "strings"

const (
	contractKeyFormat            = "tc-{address}-code"
	contractStorageItemKeyFormat = "tc-{address}-si-{key}"
)

func getContractCodeKey(addr string) string {
	key := strings.Replace(contractKeyFormat, "{address}", addr, 1)
	return key
}

func getContractStorageItemKey(addr, key string) string {
	var itemKey string
	itemKey = strings.Replace(contractStorageItemKeyFormat, "{address}", addr, 1)
	itemKey = strings.Replace(itemKey, "{key}", key, 1)
	return itemKey
}
