package httpcache

import (
	"testing"
	"time"
)

func TestRedisAdapter(t *testing.T) {
	a := NewRedisCacheAdapter(&RedisRingOptions{
		Addrs: map[string]string{
			"server": ":6379",
		},
	})

	tests := []struct {
		name     string
		key      string
		response *Response
	}{
		{
			name: "set response",
			key:  "test1",
			response: &Response{
				Value:      []byte("value 1"),
				Expiration: time.Now().Add(1 * time.Minute),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.Set(tt.key, tt.response, time.Now().Add(1*time.Minute))
		})
	}
}
