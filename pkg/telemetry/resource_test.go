package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestOtelResource(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "superplane-test")

	res, err := otelResource(context.Background())
	require.NoError(t, err)

	sdkRes, err := resource.New(context.Background(), resource.WithTelemetrySDK())
	require.NoError(t, err)
	assert.Equal(t, sdkRes.SchemaURL(), res.SchemaURL())
	assert.NotEmpty(t, res.SchemaURL())

	attrs := res.Set()
	val, ok := attrs.Value(semconv.ServiceNameKey)
	require.True(t, ok)
	assert.Equal(t, "superplane-test", val.AsString())
}
