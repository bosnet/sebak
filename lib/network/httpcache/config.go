package httpcache

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

func NewAdapter(cfg common.Config) (Adapter, error) {
	switch cfg.HTTPCacheAdapter {
	case common.HTTPCacheMemoryAdapterName:
		size := cfg.HTTPCachePoolSize
		adapter := NewMemCacheAdapter(size)
		return adapter, nil
	default:
		return nil, errors.New("adapter not found")
	}
}
