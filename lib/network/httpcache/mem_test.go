package httpcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var _ Adapter = (*MemCacheAdapter)(nil)

func TestMemCacheAdapter(t *testing.T) {
	a := NewMemCacheAdapter(10)
	now := time.Now()

	key := "key"
	resp := &Response{
		Value:      []byte("hello"),
		Expiration: now,
	}

	a.Set(key, resp, now)

	cachedResp, ok := a.Get(key)
	require.Equal(t, true, ok)
	require.Equal(t, resp, cachedResp)
}
