package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestNewMetricsResourceIncludesServiceNameFromEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "superplane-api")

	res, err := newMetricsResource(t.Context())
	require.NoError(t, err)

	name, ok := serviceNameFromResource(res)
	require.True(t, ok)
	require.Equal(t, "superplane-api", name)
}

func TestNewMetricsResourceIncludesResourceAttributesFromEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "superplane-workers")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.instance.id=pod-123")

	res, err := newMetricsResource(t.Context())
	require.NoError(t, err)

	name, ok := serviceNameFromResource(res)
	require.True(t, ok)
	require.Equal(t, "superplane-workers", name)

	instanceID, ok := attributeStringFromResource(res, "service.instance.id")
	require.True(t, ok)
	require.Equal(t, "pod-123", instanceID)
}

func serviceNameFromResource(res *resource.Resource) (string, bool) {
	return attributeStringFromResource(res, "service.name")
}

func attributeStringFromResource(res *resource.Resource, key string) (string, bool) {
	for _, attr := range res.Attributes() {
		if string(attr.Key) == key {
			return attr.Value.AsString(), true
		}
	}

	return "", false
}
