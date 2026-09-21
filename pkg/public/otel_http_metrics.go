package public

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

func resolveHTTPMetricRoute(r *http.Request) string {
	route := getOtelMetricRoute(r.Context())
	if route != "" {
		return route
	}

	if r.Pattern != "" {
		return r.Pattern
	}

	currentRoute := mux.CurrentRoute(r)
	if currentRoute != nil {
		routeTemplate, err := currentRoute.GetPathTemplate()
		if err == nil {
			return routeTemplate
		}
	}

	return ""
}

func otelHTTPMetricAttributesForRequest(r *http.Request) []attribute.KeyValue {
	/*
	 * Prefer the route resolved by grpc-gateway. Fall back to Gorilla mux for
	 * non-gateway routes that are matched directly by the outer router.
	 * Always attach superplane.request.kind so long polls do not enter
	 * the interactive API latency SLI.
	 */
	return otelHTTPMetricAttributes(resolveHTTPMetricRoute(r))
}

func otelHTTPMetricAttributes(route string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(telemetry.HTTPRequestKindAttribute, telemetry.HTTPRequestKind(route)),
	}
	if route == "" {
		return attrs
	}

	return append(attrs, attribute.String("http.route", route))
}
