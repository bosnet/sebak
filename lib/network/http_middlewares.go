package network

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	logging "github.com/inconshreveable/log15"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"github.com/ulule/limiter/drivers/store/memory"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
)

func RecoverMiddleware(logger logging.Logger) mux.MiddlewareFunc {
	if logger == nil {
		logger = log // use network.log
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("panic: %v", r)
					}
					httputils.WriteJSONError(w, err)

					logger.Error("recover an panic", "err", err)
					if VerboseLogs == true {
						debug.PrintStack()
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitReachedHandler(w http.ResponseWriter, r *http.Request) {
	httputils.WriteJSONError(w, errors.TooManyRequests)
}

func rateLimitErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	httputils.WriteJSONError(w, errors.HTTPServerError.Clone().SetData("error", err))
}

// RateLimitMiddleware throttles the incoming requests; if `Limit` is 0, there
// will be no limit.
func RateLimitMiddleware(logger logging.Logger, rule common.RateLimitRule) mux.MiddlewareFunc {
	if logger == nil {
		logger = log
	}

	store := memory.NewStoreWithOptions(
		limiter.StoreOptions{
			CleanUpInterval: time.Duration(2) * time.Minute,
		},
	)

	var defaultMiddleware *stdlib.Middleware
	if rule.Default.Limit > 0 {
		defaultMiddleware = stdlib.NewMiddleware(
			limiter.New(store, rule.Default),
			stdlib.WithForwardHeader(true),
			stdlib.WithErrorHandler(rateLimitErrorHandler),
			stdlib.WithLimitReachedHandler(rateLimitReachedHandler),
		)
	}

	middlewaresByIP := map[ /* ip address */ string]*stdlib.Middleware{}
	for ip, rate := range rule.ByIPAddress {
		var m *stdlib.Middleware
		if rate.Limit > 0 {
			m = stdlib.NewMiddleware(
				limiter.New(store, rate),
				stdlib.WithForwardHeader(true),
				stdlib.WithErrorHandler(rateLimitErrorHandler),
				stdlib.WithLimitReachedHandler(rateLimitReachedHandler),
			)
		}
		middlewaresByIP[ip] = m
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := limiter.GetIPKey(r, true)

			// from localhost, no rate limit
			if strings.HasPrefix(ip, "127.0.") {
				w.Header().Add("X-RateLimit-Limit", "")
				w.Header().Add("X-RateLimit-Remaining", "")
				w.Header().Add("X-RateLimit-Reset", "")

				next.ServeHTTP(w, r)
				return
			}

			var middleware *stdlib.Middleware

			if len(middlewaresByIP) < 1 {
				middleware = defaultMiddleware
			} else { // find middleware by ip
				var found bool
				if middleware, found = middlewaresByIP[ip]; !found {
					middleware = defaultMiddleware
				}
			}

			if middleware == nil {
				w.Header().Add("X-RateLimit-Limit", "")
				w.Header().Add("X-RateLimit-Remaining", "")
				w.Header().Add("X-RateLimit-Reset", "")

				next.ServeHTTP(w, r)
				return
			}

			context, err := middleware.Limiter.Get(r.Context(), ip)
			if err != nil {
				middleware.OnError(w, r, err)
				return
			}

			w.Header().Add("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
			w.Header().Add("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
			w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

			if context.Reached {
				middleware.OnLimitReached(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
