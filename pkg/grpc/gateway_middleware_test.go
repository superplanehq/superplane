package grpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

func TestSanitizedGatewayErrorHandler(t *testing.T) {
	t.Run("records the original error and still sanitizes the response", func(t *testing.T) {
		ctx, recorder := telemetry.WithServerErrorRecorder(context.Background())
		request := httptest.NewRequest(http.MethodGet, "/api/v1/factories", nil).WithContext(ctx)
		response := httptest.NewRecorder()

		original := errors.New("ERROR: column factories.key does not exist (SQLSTATE 42703)")
		SanitizedGatewayErrorHandler(ctx, runtime.NewServeMux(), &runtime.JSONPb{}, response, request, original)

		assert.Equal(t, original, recorder.Err())

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.NotContains(t, response.Body.String(), "factories.key")
		assert.Contains(t, response.Body.String(), sanitizedInternalMessage)
	})
}

func TestWriteGatewayHTTPError(t *testing.T) {
	t.Run("records the original error and still sanitizes the response", func(t *testing.T) {
		ctx, recorder := telemetry.WithServerErrorRecorder(context.Background())
		response := httptest.NewRecorder()

		original := errors.New("failed to load permissions: dial tcp 10.0.0.7:5432: connect: connection refused")
		writeGatewayHTTPError(ctx, response, original)

		assert.Equal(t, original, recorder.Err())

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.NotContains(t, response.Body.String(), "10.0.0.7")
		assert.Contains(t, response.Body.String(), sanitizedInternalMessage)
	})
}
