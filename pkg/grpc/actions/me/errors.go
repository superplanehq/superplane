package me

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// translateMeError classifies low-level errors returned from helpers used by
// /api/v1/me handlers so we don't surface every transient failure as an HTTP
// 500 in Sentry.
//
// The handlers in this package wrap calls to the database and the casbin-based
// authorization service. Both can fail for reasons that are not actual server
// bugs:
//
//   - context.Canceled: the client closed the connection (e.g. browser
//     navigated away while /api/v1/me was loading). Returning a 500 here
//     creates a Sentry HTTP 500 issue for what is really a benign client
//     cancellation.
//   - context.DeadlineExceeded: the per-request deadline fired before the
//     casbin enforcer / DB query returned. Surfacing this as 504 is more
//     accurate than 500.
//   - gorm.ErrRecordNotFound: the row really isn't there. 404 is correct.
//
// Anything else is preserved with the caller-supplied fallback code and
// message so genuine bugs (DB outage, casbin model load failure, ...) still
// log loudly as 500.
func translateMeError(err error, fallbackCode codes.Code, fallbackMessage string) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, "user not found")
	}

	return status.Error(fallbackCode, fallbackMessage)
}
