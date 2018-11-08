package httpcache

import (
	"time"

	redisCache "github.com/go-redis/cache"
	"github.com/go-redis/redis"
	"github.com/vmihailenco/msgpack"
)

type RedisCacheAdapter struct {
	store *redisCache.Codec
}

type RedisRingOptions redis.RingOptions

func NewRedisCacheAdapter(opt *RedisRingOptions) *RedisCacheAdapter {
	ropt := redis.RingOptions(*opt)
	a := &RedisCacheAdapter{
		&redisCache.Codec{
			Redis: redis.NewRing(&ropt),
			Marshal: func(v interface{}) ([]byte, error) {
				return msgpack.Marshal(v)
			},
			Unmarshal: func(b []byte, v interface{}) error {
				return msgpack.Unmarshal(b, v)
			},
		},
	}
	return a
}

func (a *RedisCacheAdapter) Get(key string) (*Response, bool) {
	var resp Response
	if err := a.store.Get(key, &resp); err != nil {
		return nil, false
	}
	return &resp, true
}

func (a *RedisCacheAdapter) Set(key string, resp *Response, expir time.Time) {
	var e time.Duration = 0
	if !expir.IsZero() {
		e = expir.Sub(time.Now())
	}
	a.store.Set(&redisCache.Item{
		Key:        key,
		Object:     resp,
		Expiration: e,
	})
}

func (a *RedisCacheAdapter) Remove(key string) {
	a.store.Delete(key)
}
