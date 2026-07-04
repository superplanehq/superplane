package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/superplanehq/superplane/pkg/authorization"
)

func GatewayAuthorizationMiddleware(
	muxPtr **runtime.ServeMux,
	authorizer *authorization.GatewayAuthorizer,
) runtime.Middleware {
	return func(next runtime.HandlerFunc) runtime.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			route, requiresAuth := authorizer.RouteFromRequest(r)
			if !requiresAuth {
				next(w, r.WithContext(authorization.WithPathParams(r.Context(), pathParams)), pathParams)
				return
			}

			ctx, err := authorizer.AuthorizeHTTP(r.Context(), r, route, pathParams)
			if err != nil {
				mux := *muxPtr
				if mux == nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				_, outboundMarshaler := runtime.MarshalerForRequest(mux, r)
				runtime.HTTPError(r.Context(), mux, outboundMarshaler, w, r, err)
				return
			}

			next(w, r.WithContext(ctx), pathParams)
		}
	}
}
