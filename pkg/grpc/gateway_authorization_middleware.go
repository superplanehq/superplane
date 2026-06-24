package grpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/superplanehq/superplane/pkg/authorization"
)

// GatewayAuthorizationMiddleware enforces the gateway authorization rules for
// incoming HTTP requests routed through grpc-gateway.
//
// The mux is provided through a getter (rather than as a *runtime.ServeMux
// directly) so that callers can pass it before the mux is fully constructed.
// This is required because grpc-gateway's WithMiddlewares option is evaluated
// as part of the runtime.NewServeMux(...) argument list: the local variable
// holding the mux is still nil at that point, and a direct *runtime.ServeMux
// parameter would freeze that nil value into the closure. With a getter the
// middleware reads the (now-assigned) mux at request time, on the error path
// where it has to render a gateway error response.
func GatewayAuthorizationMiddleware(
	muxGetter func() *runtime.ServeMux,
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
				mux := muxGetter()
				_, outboundMarshaler := runtime.MarshalerForRequest(mux, r)
				runtime.HTTPError(r.Context(), mux, outboundMarshaler, w, r, err)
				return
			}

			next(w, r.WithContext(ctx), pathParams)
		}
	}
}
