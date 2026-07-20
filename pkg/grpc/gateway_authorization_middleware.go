package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/superplanehq/superplane/pkg/authorization"
)

func GatewayAuthorizationMiddleware(
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
				writeGatewayHTTPError(r.Context(), w, err)
				return
			}

			next(w, r.WithContext(ctx), pathParams)
		}
	}
}
