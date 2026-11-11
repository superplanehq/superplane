package logging

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type requestLogKey struct{}

// WithRequestLogFields shares audit metadata with the outer HTTP access logger.
// Only request metadata and identifiers belong here, never bodies or credentials.
func WithRequestLogFields(ctx context.Context, fields log.Fields) context.Context {
	return context.WithValue(ctx, requestLogKey{}, fields)
}

func RequestLogFields(ctx context.Context) log.Fields {
	fields, _ := ctx.Value(requestLogKey{}).(log.Fields)
	return fields
}

// AddRequestLogFields is used synchronously by handlers before the access log.
func AddRequestLogFields(ctx context.Context, fields log.Fields) {
	logged := RequestLogFields(ctx)
	if logged == nil {
		return
	}
	for key, value := range fields {
		logged[key] = value
	}
}
