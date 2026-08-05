package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/buildinfo"
)

func VersionHeaderMiddleware(version string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(buildinfo.VersionHeader, version)
			next.ServeHTTP(w, r)
		})
	}
}
