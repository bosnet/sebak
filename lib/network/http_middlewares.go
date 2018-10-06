package network

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"boscoin.io/sebak/lib/network/httputils"
	"github.com/gorilla/mux"
)

func RecoverMiddleware(printStack bool) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("panic: %v", r)
					}
					httputils.WriteJSONError(w, err)
					log.Error("recover an panic", "err", err)
					if printStack == true {
						debug.PrintStack()
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
