package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestConfigureTestTracerProvider(t *testing.T) {
	assert.False(t, TracingEnabled())

	exporter := tracetest.NewInMemoryExporter()
	cleanup := ConfigureTestTracerProvider(exporter)
	defer cleanup()

	require.True(t, TracingEnabled())

	_, span := StartSpan(context.Background(), "test-span")
	span.End()

	require.Len(t, exporter.GetSpans(), 1)
	assert.Equal(t, "test-span", exporter.GetSpans()[0].Name)

	cleanup()
	assert.False(t, TracingEnabled())
}
