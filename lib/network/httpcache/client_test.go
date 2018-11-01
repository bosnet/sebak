package httpcache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	a := NewMemCacheAdapter(10)
	a.Set("http://foo?bar=1", &Response{
		Value: []byte("value 1"),
	}, nil)

	c, err := NewClient(
		WithAdapter(a),
	)
	require.NoError(t, err)

	cnt := 0
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("new value:%v", cnt)))
	})

	handler := c.Middleware(testHandler)

	tests := []struct {
		name   string
		url    string
		method string
		body   string
		code   int
	}{
		{
			"return cached resp",
			"http://foo?bar=1",
			"GET",
			"value 1",
			200,
		},
		{
			"return nocached resp",
			"http://foo?bar=2",
			"GET",
			"new value:2",
			200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnt++

			r, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			require.Equal(t, w.Code, tt.code)
			require.Equal(t, w.Body.String(), tt.body)
		})
	}
}
