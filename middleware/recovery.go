package middleware

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery catches panics, logs them with a stack trace, and returns 500.
func Recovery(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rv := recover(); rv != nil {
					log.Error("panic recovered",
						zap.Any("panic", rv),
						zap.ByteString("stack", debug.Stack()),
					)
					WriteError(w, Internal(nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
