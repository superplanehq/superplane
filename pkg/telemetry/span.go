package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !TracingEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}

	return Tracer().Start(ctx, name, opts...)
}

// Span starts a span and returns ctx plus a done callback for defer done(&err).
func Span(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func(*error)) {
	ctx, span := StartSpan(ctx, name, opts...)
	return ctx, func(err *error) { FinishSpan(span, err) }
}

func FinishSpan(span trace.Span, err *error) {
	if err != nil && *err != nil && shouldMarkSpanError(*err) {
		span.RecordError(*err)
		span.SetStatus(codes.Error, (*err).Error())
	}
	span.End()
}

func shouldMarkSpanError(err error) bool {
	return !isRecordNotFound(err)
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
