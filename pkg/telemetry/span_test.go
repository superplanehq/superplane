package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/gorm"
)

func TestSpanDoneRecordNotFoundDoesNotMarkSpanError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := ConfigureTestTracerProvider(exporter)
	t.Cleanup(cleanup)

	run := func() (err error) {
		_, done := Span(context.Background(), "test.find")
		defer done(&err)

		return gorm.ErrRecordNotFound
	}

	require.ErrorIs(t, run(), gorm.ErrRecordNotFound)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status.Code)
}

func TestSpanDoneWrappedRecordNotFoundDoesNotMarkSpanError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := ConfigureTestTracerProvider(exporter)
	t.Cleanup(cleanup)

	wrapped := errors.Join(errors.New("canvas lookup"), gorm.ErrRecordNotFound)
	run := func() (err error) {
		_, done := Span(context.Background(), "test.find")
		defer done(&err)

		return wrapped
	}

	require.Error(t, run())

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status.Code)
}

func TestSpanDoneOtherErrorMarksSpanError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	cleanup := ConfigureTestTracerProvider(exporter)
	t.Cleanup(cleanup)

	dbDown := errors.New("db down")
	run := func() (err error) {
		_, done := Span(context.Background(), "test.query")
		defer done(&err)

		return dbDown
	}

	require.ErrorIs(t, run(), dbDown)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
	assert.Equal(t, "db down", spans[0].Status.Description)
}
