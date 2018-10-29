package network

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/ulule/limiter"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
)

func TestRecoverMiddleware(t *testing.T) {
	handlerURL := UrlPathPrefixAPI + "/test"
	panicMsg := "Don't panic,just use go"
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(panicMsg)
	}

	router := mux.NewRouter()
	router.Use(RecoverMiddleware(nil))
	router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + handlerURL)
	require.NoError(t, err)

	require.Equal(t, 500, resp.StatusCode)
	require.Equal(t, "application/problem+json", resp.Header["Content-Type"][0])

	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)

	var msg map[string]interface{}
	err = json.Unmarshal(bs, &msg)
	require.NoError(t, err)
	require.Equal(t, "panic: "+panicMsg, msg["title"])
}

func testRequestForRateLimit(ts *httptest.Server, u, ip string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", ts.URL+u, nil)
	req.Header.Set("X-Forwarded-For", ip)
	return ts.Client().Do(req)
}

func TestRateLimitMiddleWare(t *testing.T) {
	handlerURL := UrlPathPrefixAPI + "/test"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1"))
		return
	}

	{ // 1000 requests per second
		rate := limiter.Rate{
			Period: 1 * time.Second,
			Limit:  1000,
		}
		router := mux.NewRouter()
		router.Use(RateLimitMiddleware(nil, common.NewRateLimitRule(rate)))
		router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")
		ts := httptest.NewServer(router)

		resp, err := testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
		require.NoError(t, err)
		ts.Close()

		require.Equal(t, fmt.Sprintf("%d", rate.Limit), resp.Header.Get("X-Ratelimit-Limit"))
		require.Equal(t, fmt.Sprintf("%d", rate.Limit-1), resp.Header.Get("X-Ratelimit-Remaining"))
		require.NotEmptyf(t, resp.Header.Get("X-Ratelimit-Reset"), "")

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		require.Equal(t, []byte("1"), body)
	}

	{ // 1 requests per minute; over rate limit, the `HttpProblem` will be
		// returned with `http.StatusTooManyRequests` status code
		rate := limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  1,
		}
		router := mux.NewRouter()
		router.Use(RateLimitMiddleware(nil, common.NewRateLimitRule(rate)))
		router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")
		ts := httptest.NewServer(router)

		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
				wg.Done()
			}()
		}
		wg.Wait()

		resp, err := testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
		require.NoError(t, err)

		require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

		require.Equal(t, fmt.Sprintf("%d", rate.Limit), resp.Header.Get("X-Ratelimit-Limit"))
		require.Equal(t, "0", resp.Header.Get("X-Ratelimit-Remaining"))
		require.NotEmptyf(t, resp.Header.Get("X-Ratelimit-Reset"), "")

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var problem httputils.Problem
		{
			err := json.Unmarshal(body, &problem)
			require.NoError(t, err)
		}
		require.Equal(
			t,
			problem.Type,
			httputils.ProblemTypeByCode(errors.TooManyRequests.Code),
		)

		ts.Close()
	}
}

func TestRateLimitMiddleWareByIPAddress(t *testing.T) {
	handlerURL := UrlPathPrefixAPI + "/test"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1"))
		return
	}

	allowedIP := "1.1.1.1"
	// by default, 1 requests per minute, but 1000 request per minute from `1.1.1.1`
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  1,
	}
	rate1000 := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  1000,
	}

	rule := common.NewRateLimitRule(rate)
	rule.ByIPAddress[allowedIP] = rate1000

	router := mux.NewRouter()
	router.Use(RateLimitMiddleware(nil, rule))
	router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")
	ts := httptest.NewServer(router)

	{ // from localhost
		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
				wg.Done()
			}()
		}
		wg.Wait()

		resp, err := testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
		require.NoError(t, err)

		require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	}

	{ // from localhost
		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				req, _ := http.NewRequest("GET", ts.URL+handlerURL, nil)
				req.Header.Set("X-Forwarded-For", allowedIP)
				ts.Client().Do(req)
				wg.Done()
			}()
		}
		wg.Wait()

		req, _ := http.NewRequest("GET", ts.URL+handlerURL, nil)
		req.Header.Set("X-Forwarded-For", allowedIP)
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		require.Equal(t, []byte("1"), body)
	}
	ts.Close()
}

func TestRateLimitMiddleWareUnlimit(t *testing.T) {
	handlerURL := UrlPathPrefixAPI + "/test"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1"))
		return
	}

	{ // set `Limit` to 0, unlimited requests
		rate := limiter.Rate{
			Period: 1 * time.Second,
			Limit:  0,
		}
		router := mux.NewRouter()
		router.Use(RateLimitMiddleware(nil, common.NewRateLimitRule(rate)))
		router.HandleFunc(handlerURL, http.HandlerFunc(handler)).Methods("GET")
		ts := httptest.NewServer(router)

		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				testRequestForRateLimit(ts, handlerURL, "3.3.3.")
				wg.Done()
			}()
		}
		wg.Wait()

		resp, err := testRequestForRateLimit(ts, handlerURL, "3.3.3.3")
		require.NoError(t, err)
		ts.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// under unlimited rate limit, the rate limit related headers will be
		// empty.
		require.Emptyf(t, resp.Header.Get("X-Ratelimit-Limit"), "")
		require.Emptyf(t, resp.Header.Get("X-Ratelimit-Remaining"), "")
		require.Emptyf(t, resp.Header.Get("X-Ratelimit-Reset"), "")

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		require.Equal(t, []byte("1"), body)
	}
}
