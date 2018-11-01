package httpcache

import (
	"time"

	"github.com/hashicorp/golang-lru"
)

type MemCacheAdapter struct {
	lruCache *lru.Cache
}

func NewMemCacheAdapter(size int) *MemCacheAdapter {
	lruCache, err := lru.New(size)
	if err != nil {
		panic(err)
	}

	a := &MemCacheAdapter{
		lruCache: lruCache,
	}
	return a
}

func (a *MemCacheAdapter) Get(key string) (*Response, bool) {
	value, ok := a.lruCache.Get(key)
	if ok {
		res, ok := value.(*Response)
		return res, ok
	}
	return nil, ok
}

func (a *MemCacheAdapter) Set(key string, resp *Response, expir *time.Time) {
	a.lruCache.Add(key, resp)
}

func (a *MemCacheAdapter) Remove(key string) {
	a.lruCache.Remove(key)
}
