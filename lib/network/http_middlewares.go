package network

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	logging "github.com/inconshreveable/log15"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"github.com/ulule/limiter/drivers/store/memory"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
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
	httputils.WriteJSONError(w, errors.ErrorTooManyRequests)
}

func rateLimitErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	httputils.WriteJSONError(w, errors.ErrorHTTPServerError.Clone().SetData("error", err))
}

func RateLimitMiddleware(logger logging.Logger, rule common.RateLimitRule) mux.MiddlewareFunc {
	if logger == nil {
		logger = log
	}

	store := memory.NewStoreWithOptions(
		limiter.StoreOptions{
			CleanUpInterval: time.Duration(2) * time.Minute,
		},
	)

	defaultMiddleware := stdlib.NewMiddleware(
		limiter.New(store, rule.Default),
		stdlib.WithForwardHeader(true),
		stdlib.WithErrorHandler(rateLimitErrorHandler),
		stdlib.WithLimitReachedHandler(rateLimitReachedHandler),
	)

	middlewares := map[ /* ip address */ string]*stdlib.Middleware{}
	for ip, rate := range rule.ByIPAddress {
		m := stdlib.NewMiddleware(
			limiter.New(store, rate),
			stdlib.WithForwardHeader(true),
			stdlib.WithErrorHandler(rateLimitErrorHandler),
			stdlib.WithLimitReachedHandler(rateLimitReachedHandler),
		)
		middlewares[ip] = m
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// find middleware by ip
			ip := limiter.GetIPKey(r, true)

			var middleware *stdlib.Middleware
			var found bool
			if middleware, found = middlewares[ip]; !found {
				middleware = defaultMiddleware
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
