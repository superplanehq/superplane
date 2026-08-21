package grpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/requesterror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestSanitizedGatewayErrorHandler(t *testing.T) {
	t.Run("records the cause of a server error", func(t *testing.T) {
		ctx, recorder := requesterror.NewContext(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/api/v1/factories", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		cause := errors.New("column factories.next_work_order_number does not exist")

		SanitizedGatewayErrorHandler(r.Context(), runtime.NewServeMux(), &runtime.JSONPb{}, w, r, cause)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, cause, recorder.Err())
	})

	t.Run("keeps the response sanitized", func(t *testing.T) {
		ctx, _ := requesterror.NewContext(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/api/v1/factories", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		SanitizedGatewayErrorHandler(
			r.Context(),
			runtime.NewServeMux(),
			&runtime.JSONPb{},
			w,
			r,
			errors.New("column factories.next_work_order_number does not exist"),
		)

		assert.Contains(t, w.Body.String(), sanitizedInternalMessage)
		assert.NotContains(t, w.Body.String(), "next_work_order_number")
	})

	t.Run("does not record a client error", func(t *testing.T) {
		ctx, recorder := requesterror.NewContext(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/api/v1/factories/unknown", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		SanitizedGatewayErrorHandler(r.Context(), runtime.NewServeMux(), &runtime.JSONPb{}, w, r, gorm.ErrRecordNotFound)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, recorder.Err())
	})

	t.Run("does not panic without a recorder in the context", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/factories", nil)
		w := httptest.NewRecorder()

		require.NotPanics(t, func() {
			SanitizedGatewayErrorHandler(
				r.Context(),
				runtime.NewServeMux(),
				&runtime.JSONPb{},
				w,
				r,
				errors.New("db down"),
			)
		})
	})
}

func TestWriteGatewayHTTPError(t *testing.T) {
	t.Run("records the cause of a server error", func(t *testing.T) {
		ctx, recorder := requesterror.NewContext(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		cause := errors.New("permission check failed")

		writeGatewayHTTPError(r, w, cause)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, cause, recorder.Err())
	})

	t.Run("does not record a denied request", func(t *testing.T) {
		ctx, recorder := requesterror.NewContext(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		writeGatewayHTTPError(r, w, status.Error(codes.NotFound, "Not found"))

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, recorder.Err())
	})
}

func TestIsServerError(t *testing.T) {
	t.Run("reports server errors", func(t *testing.T) {
		assert.True(t, isServerError(status.Error(codes.Internal, "internal error")))
		assert.True(t, isServerError(status.Error(codes.Unknown, "unknown")))
		assert.True(t, isServerError(status.Error(codes.Unavailable, "unavailable")))
	})

	t.Run("reports client errors as no server error", func(t *testing.T) {
		assert.False(t, isServerError(status.Error(codes.NotFound, "resource not found")))
		assert.False(t, isServerError(status.Error(codes.InvalidArgument, "bad request")))
		assert.False(t, isServerError(status.Error(codes.PermissionDenied, "denied")))
		assert.False(t, isServerError(nil))
	})
}
