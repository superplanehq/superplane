package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/superplanehq/superplane/pkg/authorization"
	"google.golang.org/grpc/status"
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
				sanitized := SanitizeError(r.Context(), err)
				st, ok := status.FromError(sanitized)
				if !ok {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				httpCode := runtime.HTTPStatusFromCode(st.Code())
				http.Error(w, st.Message(), httpCode)
				return
			}

			next(w, r.WithContext(ctx), pathParams)
		}
	}
}
