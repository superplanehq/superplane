package grpc

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
)

func GatewayRecoveryMiddleware() runtime.Middleware {
	return func(next runtime.HandlerFunc) runtime.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			defer func() {
				if recovered := recover(); recovered != nil {
					_ = sentryRecoveryHandler(recovered)
					log.Errorf("recovered from panic in grpc-gateway handler: %v. Stack: %s", recovered, debug.Stack())
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next(w, r, pathParams)
		}
	}
}

func SanitizedGatewayErrorHandler(
	ctx context.Context,
	mux *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, SanitizeError(r.Context(), err))
}
