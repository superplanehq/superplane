package grpc

import (
	"context"
	"errors"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/status"
)

type serverErrorReportKey struct{}

// serverErrorReport is filled by the gateway after LoggingMiddleware attaches it.
// The pointer stays on the request context so the access log can read a later write.
type serverErrorReport struct {
	cause    error
	source   error
	reported bool
}

// NotedServerError is a server-error cause kept for the error tracker and the access log.
// Source is the original handler error, when one was present.
type NotedServerError struct {
	Cause  error
	Source error
}

// WithServerErrorReport attaches an empty holder to the request.
// Later writes stay visible after the handler returns.
func WithServerErrorReport(r *http.Request) *http.Request {
	if r == nil || serverErrorReportFrom(r.Context()) != nil {
		return r
	}

	return r.WithContext(context.WithValue(r.Context(), serverErrorReportKey{}, &serverErrorReport{}))
}

// NotedServerErrorFrom returns the cause stored for this request.
func NotedServerErrorFrom(ctx context.Context) (NotedServerError, bool) {
	report := serverErrorReportFrom(ctx)
	if report == nil || report.cause == nil {
		return NotedServerError{}, false
	}

	return NotedServerError{Cause: report.cause, Source: report.source}, true
}

// ServerErrorAlreadyReported reports whether a panic was already sent to the error tracker.
func ServerErrorAlreadyReported(ctx context.Context) bool {
	report := serverErrorReportFrom(ctx)
	return report != nil && report.reported
}

// ReportableHTTPStatus reports whether a response status is a server error
// worth forwarding to the error tracker.
//
// Only true server errors (5xx) qualify. Some 5xx codes mean the client sent
// an unsupported request, not that the server failed:
//
//   - 501 Not Implemented is returned by grpc-gateway when a path exists but
//     the HTTP method has no mapping (for example POST /api/v1/triggers/start
//     when only GET /api/v1/triggers/{name} is defined). Those requests should
//     not create issues.
//   - 505 HTTP Version Not Supported is likewise a client-caused mismatch.
func ReportableHTTPStatus(httpStatus int) bool {
	if httpStatus < http.StatusInternalServerError {
		return false
	}

	switch httpStatus {
	case http.StatusNotImplemented, http.StatusHTTPVersionNotSupported:
		return false
	}

	return true
}

func noteReportableServerError(ctx context.Context, err error, sanitized error) {
	if err == nil || sanitized == nil {
		return
	}

	httpStatus := runtime.HTTPStatusFromCode(status.Code(sanitized))
	if !ReportableHTTPStatus(httpStatus) {
		return
	}

	cause := errors.Unwrap(err)
	safeMessage := status.Convert(sanitized).Message()
	if !notableCause(cause, safeMessage) {
		return
	}

	report := serverErrorReportFrom(ctx)
	if report == nil {
		return
	}

	report.cause = cause
	report.source = err
}

func notableCause(cause error, safeMessage string) bool {
	if cause == nil {
		return false
	}

	message := cause.Error()
	return message != "" && message != safeMessage
}

func markServerErrorReported(ctx context.Context) {
	report := serverErrorReportFrom(ctx)
	if report == nil {
		return
	}

	report.reported = true
}

func serverErrorReportFrom(ctx context.Context) *serverErrorReport {
	if ctx == nil {
		return nil
	}

	report, _ := ctx.Value(serverErrorReportKey{}).(*serverErrorReport)
	return report
}
