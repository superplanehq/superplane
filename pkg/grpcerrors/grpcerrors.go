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

// handlerError carries a client-safe message and gRPC code while preserving the
// original error for classification (context cancel, etc.) at the gateway.
type handlerError struct {
	code    codes.Code
	message string
	err     error
}

func (e *handlerError) Error() string {
	return e.message
}

func (e *handlerError) Unwrap() error {
	return e.err
}

func newHandlerError(err error, message string, code codes.Code) error {
	if err == nil {
		return nil
	}

	return &handlerError{
		code:    code,
		message: message,
		err:     err,
	}
}

// Internal wraps err as an internal server error for the grpc-gateway sanitizer.
func Internal(err error, message string) error {
	return newHandlerError(err, message, codes.Internal)
}

// NotFound wraps err as a not-found error for the grpc-gateway sanitizer.
func NotFound(err error, message string) error {
	return newHandlerError(err, message, codes.NotFound)
}

// HandlerStatus returns the gRPC code and client-safe message from a handler error.
func HandlerStatus(err error) (codes.Code, string, bool) {
	var wrapped *handlerError
	if !errors.As(err, &wrapped) {
		return codes.OK, "", false
	}

	return wrapped.code, wrapped.message, true
}

// HandlerMessage returns the client-safe message from a handler error.
func HandlerMessage(err error) (string, bool) {
	_, message, ok := HandlerStatus(err)
	return message, ok
}
