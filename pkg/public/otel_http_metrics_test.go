package public

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

func TestOtelHTTPMetricAttributes(t *testing.T) {
	t.Run("labels interactive API routes", func(t *testing.T) {
		attrs := otelHTTPMetricAttributes("/api/v1/integrations")
		require.Equal(t, []attribute.KeyValue{
			attribute.String(telemetry.HTTPRequestKindAttribute, telemetry.HTTPRequestKindAPI),
			attribute.String("http.route", "/api/v1/integrations"),
		}, attrs)
	})

	t.Run("labels planning wait as long poll", func(t *testing.T) {
		attrs := otelHTTPMetricAttributes("/api/v1/runner/planning-sessions/wait")
		require.Equal(t, []attribute.KeyValue{
			attribute.String(telemetry.HTTPRequestKindAttribute, telemetry.HTTPRequestKindLongPoll),
			attribute.String("http.route", "/api/v1/runner/planning-sessions/wait"),
		}, attrs)
	})

	t.Run("labels webhook routes", func(t *testing.T) {
		attrs := otelHTTPMetricAttributes("/api/v1/webhooks/{webhookID}")
		require.Equal(t, []attribute.KeyValue{
			attribute.String(telemetry.HTTPRequestKindAttribute, telemetry.HTTPRequestKindWebhook),
			attribute.String("http.route", "/api/v1/webhooks/{webhookID}"),
		}, attrs)
	})

	t.Run("keeps request kind when the route template is missing", func(t *testing.T) {
		attrs := otelHTTPMetricAttributes("")
		require.Equal(t, []attribute.KeyValue{
			attribute.String(telemetry.HTTPRequestKindAttribute, telemetry.HTTPRequestKindAPI),
		}, attrs)
	})
}

func TestResolveHTTPMetricRoutePrefersGatewayTemplate(t *testing.T) {
	req := httptestRequestWithRoute("/api/v1/organizations/{id}")
	assert.Equal(t, "/api/v1/organizations/{id}", resolveHTTPMetricRoute(req))
}

func httptestRequestWithRoute(pattern string) *http.Request {
	req := (&http.Request{}).WithContext((&http.Request{}).Context())
	req = withOtelMetricRoute(req)
	setOtelMetricRoute(req.Context(), pattern)
	return req
}
