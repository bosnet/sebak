package network

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/network/httputils"
)

func RecoverMiddleware(logger log15.Logger) mux.MiddlewareFunc {
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
