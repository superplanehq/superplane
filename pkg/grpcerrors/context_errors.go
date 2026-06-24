package grpcerrors

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	RequestCanceledMessage         = "request canceled"
	RequestDeadlineExceededMessage = "request deadline exceeded"
)

// StatusFromContextError maps context cancellation errors to gRPC status codes
// suitable for HTTP translation via grpc-gateway:
//
//   - context.Canceled (typically client disconnect) -> codes.Canceled (HTTP 499)
//   - context.DeadlineExceeded -> codes.DeadlineExceeded (HTTP 504)
//
// requestCtx should be the incoming HTTP/gRPC request context. When the client
// disconnects, requestCtx.Err() is set and Canceled is returned — see Brandur's
// notes on distinguishing local vs request context:
// https://brandur.org/fragments/testing-request-cancellation
func StatusFromContextError(requestCtx context.Context, err error) (ok bool, statusErr error) {
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return false, nil
	}

	if requestCtx.Err() != nil && errors.Is(requestCtx.Err(), context.Canceled) {
		return true, status.Error(codes.Canceled, RequestCanceledMessage)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true, status.Error(codes.DeadlineExceeded, RequestDeadlineExceededMessage)
	}

	if errors.Is(err, context.Canceled) {
		return true, status.Error(codes.Canceled, RequestCanceledMessage)
	}

	return false, nil
}
