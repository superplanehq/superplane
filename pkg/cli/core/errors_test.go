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

func TestFormatGoogleRPCStatusErrorReturnsNilForUnknownMessage(t *testing.T) {
	message := "something else"
	status := openapi_client.GooglerpcStatus{Message: &message}

	err := formatGoogleRPCStatusError(&status)
	require.NoError(t, err)
}
