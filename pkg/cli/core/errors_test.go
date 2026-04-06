package core

import (
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
