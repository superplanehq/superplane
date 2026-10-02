package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestLoggingMiddleware_StoresWrappedCauseThroughGatewayRoute(t *testing.T) {
	const (
		path        = "/api/v1/server-error-cause/1"
		safeMessage = "failed to describe work order"
		causeText   = "db down"
	)

	transport := bindCaptureTransport(t)
	logger, buffer := newJSONLogger()
	mux := gatewayRouteMux()
	require.NoError(t, mux.HandlePath(http.MethodGet, "/api/v1/server-error-cause/{id}", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		forwardHandlerError(mux, w, r, grpcerrors.Internal(errors.New(causeText), safeMessage))
	}))

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloned := CloneGatewayRequest(r)
		TraceGatewayServe(r.Context(), w, mux, cloned.WithContext(r.Context()))
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), safeMessage)
	assert.NotContains(t, recorder.Body.String(), causeText)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "error", payload["level"])
	assert.Equal(t, causeText, payload["error"])

	events := transport.Events()
	require.Len(t, events, 1)
	event := events[0]
	assert.Equal(t, []string{causeText}, exceptionValues(event))
	assert.NotContains(t, event.Message, statusReport(http.StatusInternalServerError, path))
	assert.Equal(t, "500", event.Tags["status"])
	assert.Equal(t, safeMessage, event.Extra["handler_message"])
}

func TestLoggingMiddleware_StoresWrappedServerErrorCause(t *testing.T) {
	const (
		path        = "/api/v1/factories/orders/1"
		safeMessage = "failed to describe work order"
		causeText   = "db down"
	)

	transport := bindCaptureTransport(t)
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grpc.SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			gatewayJSONMarshaler(),
			w,
			r,
			grpcerrors.Internal(errors.New(causeText), safeMessage),
		)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), safeMessage)
	assert.NotContains(t, recorder.Body.String(), causeText)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "error", payload["level"])
	assert.Equal(t, causeText, payload["error"])

	events := transport.Events()
	require.Len(t, events, 1)
	event := events[0]
	assert.Equal(t, []string{causeText}, exceptionValues(event))
	assert.NotContains(t, event.Message, statusReport(http.StatusInternalServerError, path))
	assert.Equal(t, "500", event.Tags["status"])
	assert.Equal(t, safeMessage, event.Extra["handler_message"])
}

func TestLoggingMiddleware_DoesNotStoreCanceledRequest(t *testing.T) {
	transport := bindCaptureTransport(t)
	logger, _ := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grpc.SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			gatewayJSONMarshaler(),
			w,
			r,
			grpcerrors.Internal(context.Canceled, "failed to describe work order"),
		)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/factories/orders/1", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, 499, recorder.Code)
	assert.Empty(t, transport.Events())
}

func TestLoggingMiddleware_StoresStatusMessageWithoutCause(t *testing.T) {
	const path = "/api/v1/factories/orders/1"

	transport := bindCaptureTransport(t)
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grpc.SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			gatewayJSONMarshaler(),
			w,
			r,
			errors.New("db down"),
		)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "internal error")
	assert.NotContains(t, recorder.Body.String(), "db down")

	payload := decodeLogLine(t, buffer.String())
	_, hasError := payload["error"]
	assert.False(t, hasError)

	events := transport.Events()
	require.Len(t, events, 1)
	assert.Equal(t, statusReport(http.StatusInternalServerError, path), events[0].Message)
	assert.Empty(t, events[0].Exception)
}

func TestLoggingMiddleware_DoesNotStoreStatusMessageAfterPanic(t *testing.T) {
	const path = "/api/v1/factories/orders/1"

	transport := bindCaptureTransport(t)
	logger, _ := newJSONLogger()
	recovered := grpc.GatewayRecoveryMiddleware()(func(http.ResponseWriter, *http.Request, map[string]string) {
		panic("handler panicked")
	})
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recovered(w, r, nil)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	events := transport.Events()
	require.NotEmpty(t, events)
	for _, event := range events {
		assert.NotContains(t, event.Message, statusReport(http.StatusInternalServerError, path))
	}
}

func TestLoggingMiddleware_StoresStatusMessageWhenCauseRepeatsSafeMessage(t *testing.T) {
	const (
		path        = "/api/v1/factories/orders/1"
		safeMessage = "failed to describe work order"
	)

	transport := bindCaptureTransport(t)
	logger, _ := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grpc.SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			gatewayJSONMarshaler(),
			w,
			r,
			grpcerrors.Internal(errors.New(safeMessage), safeMessage),
		)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), safeMessage)

	events := transport.Events()
	require.Len(t, events, 1)
	assert.Equal(t, statusReport(http.StatusInternalServerError, path), events[0].Message)
	assert.Empty(t, events[0].Exception)
}

func TestLoggingMiddleware_StoresHandlerServerErrorCause(t *testing.T) {
	const (
		path      = "/api/v1/runner/planning-sessions/confidence"
		causeText = "scan recent_scores: sql: Scan error"
		safeBody  = "Planning session failed\n"
	)

	transport := bindCaptureTransport(t)
	logger, _ := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetServerError(r.Context(), errors.New(causeText), map[string]string{
			"route":               path,
			"planning_session_id": "session-1",
		})
		http.Error(w, "Planning session failed", http.StatusInternalServerError)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, safeBody, recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), causeText)

	events := transport.Events()
	require.Len(t, events, 1)
	event := events[0]
	assert.Empty(t, event.Message)
	assert.Equal(t, []string{causeText}, exceptionValues(event))
	assert.NotContains(t, event.Message, statusReport(http.StatusInternalServerError, path))
	assert.Equal(t, "500", event.Tags["status"])
	assert.Equal(t, path, event.Tags["route"])
	assert.Equal(t, "session-1", event.Tags["planning_session_id"])
}

func TestLoggingMiddleware_DoesNotStoreClientError(t *testing.T) {
	transport := bindCaptureTransport(t)
	logger, _ := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grpc.SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			gatewayJSONMarshaler(),
			w,
			r,
			status.Error(codes.NotFound, "work order not found"),
		)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/factories/orders/1", nil))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, transport.Events())
}

func gatewayRouteMux() *runtime.ServeMux {
	return runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, gatewayJSONMarshaler()),
		runtime.WithErrorHandler(grpc.SanitizedGatewayErrorHandler),
		runtime.WithMiddlewares(
			grpc.GatewayRecoveryMiddleware(),
			grpc.GatewayAuthorizationMiddleware(authorization.NewGatewayAuthorizer(nil)),
		),
	)
}

func forwardHandlerError(mux *runtime.ServeMux, w http.ResponseWriter, r *http.Request, err error) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	_, outbound := runtime.MarshalerForRequest(mux, r)
	annotated, annotateErr := runtime.AnnotateIncomingContext(
		ctx,
		mux,
		r,
		"/superplane.factories.Factories/DescribeWorkOrder",
		runtime.WithHTTPPathPattern("/api/v1/factories/{factory_id}/orders/{order_id}"),
	)
	if annotateErr != nil {
		runtime.HTTPError(ctx, mux, outbound, w, r, annotateErr)
		return
	}

	annotated = runtime.NewServerMetadataContext(annotated, runtime.ServerMetadata{})
	runtime.HTTPError(annotated, mux, outbound, w, r, err)
}

func gatewayJSONMarshaler() runtime.Marshaler {
	return &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true},
	}
}

func statusReport(status int, path string) string {
	return fmt.Sprintf("HTTP %d %s", status, path)
}

func exceptionValues(event *sentry.Event) []string {
	values := make([]string, 0, len(event.Exception))
	for _, exception := range event.Exception {
		values = append(values, exception.Value)
	}
	return values
}

func bindCaptureTransport(t *testing.T) *captureTransport {
	t.Helper()

	transport := &captureTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@localhost/1",
		Transport: transport,
	})
	require.NoError(t, err)

	hub := sentry.CurrentHub()
	previous := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() {
		hub.BindClient(previous)
	})
	return transport
}

type captureTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *captureTransport) Configure(sentry.ClientOptions) {}

func (t *captureTransport) Flush(time.Duration) bool { return true }

func (t *captureTransport) SendEvent(event *sentry.Event) {
	if event == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	clone := *event
	t.events = append(t.events, &clone)
}

func (t *captureTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}
