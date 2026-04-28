package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestFormatGoogleRPCStatusErrorForUsageLimit(t *testing.T) {
	message := "organization canvas limit exceeded"
	status := openapi_client.GooglerpcStatus{Message: &message}

	err := formatGoogleRPCStatusError(&status)
	require.Error(t, err)
	require.Equal(
		t,
		"usage limit reached: this organization has reached its canvas limit\nSee current limits with: superplane usage get",
		err.Error(),
	)
}

func TestFormatGoogleRPCStatusErrorSurfacesUnknownMessage(t *testing.T) {
	message := "something else"
	status := openapi_client.GooglerpcStatus{Message: &message}

	err := formatGoogleRPCStatusError(&status)
	require.Error(t, err)
	require.Equal(t, "something else", err.Error())
}

func TestFormatGoogleRPCStatusErrorWithGRPCCodePrefix(t *testing.T) {
	tests := []struct {
		name     string
		code     int32
		message  string
		expected string
	}{
		{
			name:     "InvalidArgument",
			code:     3,
			message:  "secret not found",
			expected: "invalid request: secret not found",
		},
		{
			name:     "NotFound",
			code:     5,
			message:  "canvas not found",
			expected: "not found: canvas not found",
		},
		{
			name:     "AlreadyExists",
			code:     6,
			message:  "resource already exists",
			expected: "already exists: resource already exists",
		},
		{
			name:     "PermissionDenied",
			code:     7,
			message:  "access denied",
			expected: "permission denied: access denied",
		},
		{
			name:     "Unimplemented",
			code:     12,
			message:  "method not implemented",
			expected: "not supported: method not implemented",
		},
		{
			name:     "Internal",
			code:     13,
			message:  "unexpected failure",
			expected: "internal error: unexpected failure",
		},
		{
			name:     "Unavailable",
			code:     14,
			message:  "service is down",
			expected: "service unavailable: service is down",
		},
		{
			name:     "Unauthenticated",
			code:     16,
			message:  "user not authenticated",
			expected: "authentication required: user not authenticated",
		},
		{
			name:     "UnknownCode",
			code:     99,
			message:  "some error",
			expected: "some error",
		},
		{
			name:     "ZeroCode",
			code:     0,
			message:  "some error",
			expected: "some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := tt.code
			status := openapi_client.GooglerpcStatus{
				Code:    &code,
				Message: &tt.message,
			}

			err := formatGoogleRPCStatusError(&status)
			require.Error(t, err)
			require.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestFormatGoogleRPCStatusErrorReturnsNilForEmptyMessage(t *testing.T) {
	message := ""
	code := int32(3)
	status := openapi_client.GooglerpcStatus{Message: &message, Code: &code}

	err := formatGoogleRPCStatusError(&status)
	require.NoError(t, err)
}

func TestFormatGoogleRPCStatusErrorReturnsNilForNilStatus(t *testing.T) {
	err := formatGoogleRPCStatusError(nil)
	require.NoError(t, err)
}

func TestFormatGoogleRPCStatusErrorUsageLimitTakesPrecedence(t *testing.T) {
	message := "organization canvas limit exceeded"
	code := int32(3)
	status := openapi_client.GooglerpcStatus{Message: &message, Code: &code}

	err := formatGoogleRPCStatusError(&status)
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage limit reached")
}

func TestFormatCommandErrorPassesThroughNonAPIErrors(t *testing.T) {
	plain := fmt.Errorf("boom")
	require.Equal(t, plain, FormatCommandError(plain))
	require.NoError(t, FormatCommandError(nil))
}

func TestFormatCommandErrorDecodesValidationBody(t *testing.T) {
	body := map[string]any{
		"code":    3,
		"message": "canvas name is required",
	}
	err := runAPICallWithResponse(t, http.StatusBadRequest, "application/json", mustJSON(t, body))

	require.Equal(t, "invalid request: canvas name is required", FormatCommandError(err).Error())
}

func TestFormatCommandErrorAppendsFieldViolations(t *testing.T) {
	body := map[string]any{
		"code":    3,
		"message": "canvas is invalid",
		"details": []any{
			map[string]any{
				"@type": "type.googleapis.com/google.rpc.BadRequest",
				"fieldViolations": []any{
					map[string]any{"field": "canvas.name", "description": "must not be empty"},
					map[string]any{"field": "canvas.description", "description": "must be under 200 chars"},
				},
			},
		},
	}
	err := runAPICallWithResponse(t, http.StatusBadRequest, "application/json", mustJSON(t, body))

	formatted := FormatCommandError(err)
	require.Error(t, formatted)
	require.Equal(t,
		"invalid request: canvas is invalid\n  - canvas.name: must not be empty\n  - canvas.description: must be under 200 chars",
		formatted.Error(),
	)
}

func TestFormatCommandErrorFallsBackToBodyForNonJSONResponse(t *testing.T) {
	err := runAPICallWithResponse(t, http.StatusBadRequest, "text/plain", []byte("upstream validation failed: name must not be empty"))

	formatted := FormatCommandError(err)
	require.Error(t, formatted)
	require.Contains(t, formatted.Error(), "upstream validation failed: name must not be empty")
}

func TestFormatCommandErrorHandlesEmptyBody(t *testing.T) {
	err := runAPICallWithResponse(t, http.StatusBadRequest, "", nil)

	formatted := FormatCommandError(err)
	require.Error(t, formatted)
	require.Contains(t, formatted.Error(), "400 Bad Request")
}

func TestFormatCommandErrorHandlesZeroCodeBody(t *testing.T) {
	body := map[string]any{
		"message": "something broke",
	}
	err := runAPICallWithResponse(t, http.StatusBadRequest, "application/json", mustJSON(t, body))

	require.Equal(t, "something broke", FormatCommandError(err).Error())
}

func TestFormatCommandErrorTruncatesLargeBodies(t *testing.T) {
	big := strings.Repeat("x", maxErrorBodyBytes+500)
	err := runAPICallWithResponse(t, http.StatusBadRequest, "text/plain", []byte(big))

	formatted := FormatCommandError(err).Error()
	require.Contains(t, formatted, "... [truncated]")
	require.Less(t, len(formatted), maxErrorBodyBytes+200)
}

func runAPICallWithResponse(t *testing.T, statusCode int, contentType string, body []byte) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(statusCode)
		if len(body) > 0 {
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(server.Close)

	cfg := openapi_client.NewConfiguration()
	cfg.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}
	client := openapi_client.NewAPIClient(cfg)

	_, _, err := client.CanvasAPI.CanvasesDescribeCanvas(context.Background(), "canvas-id").Execute()
	require.Error(t, err)
	return err
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
