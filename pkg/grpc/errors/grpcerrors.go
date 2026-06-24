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

func handlerErrorWithCode(err error, message string, code codes.Code) error {
	if err == nil {
		err = errors.New(message)
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

// InvalidArgument wraps err as an invalid-argument error for the grpc-gateway sanitizer.
func InvalidArgument(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.InvalidArgument)
}

// Unauthenticated wraps err as an unauthenticated error for the grpc-gateway sanitizer.
func Unauthenticated(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.Unauthenticated)
}

// PermissionDenied wraps err as a permission-denied error for the grpc-gateway sanitizer.
func PermissionDenied(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.PermissionDenied)
}

// AlreadyExists wraps err as an already-exists error for the grpc-gateway sanitizer.
func AlreadyExists(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.AlreadyExists)
}

// FailedPrecondition wraps err as a failed-precondition error for the grpc-gateway sanitizer.
func FailedPrecondition(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.FailedPrecondition)
}

// ResourceExhausted wraps err as a resource-exhausted error for the grpc-gateway sanitizer.
func ResourceExhausted(err error, message string) error {
	return handlerErrorWithCode(err, message, codes.ResourceExhausted)
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

// Code returns the gRPC code from a handler or status error.
func Code(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	if code, _, ok := HandlerStatus(err); ok {
		return code
	}

	return status.Code(err)
}
