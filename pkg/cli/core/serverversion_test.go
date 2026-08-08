package core

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestVersionSkewHintForMismatchedVersions(t *testing.T) {
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.26.0", nil
		},
	})

	require.Contains(t, hint, "v0.30.0")
	require.Contains(t, hint, "v0.26.0")
	require.Contains(t, hint, "matching CLI release")
}

func TestVersionSkewHintWhenServerVersionUnavailable(t *testing.T) {
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "", ErrServerVersionUnavailable
		},
	})

	require.Contains(t, hint, "does not report its version")
	require.Contains(t, hint, "v0.30.0")
}

func TestVersionSkewHintSilentWhenServerIsNewer(t *testing.T) {
	// A server newer than the CLI is not the version-skew case this hint
	// covers: an older CLI predates the endpoint but a "not found" there is
	// far more likely a genuine routing issue, not skew.
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.26.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.30.0", nil
		},
	})

	require.Empty(t, hint)
}

func TestVersionSkewHintSilentWhenVersionsMatch(t *testing.T) {
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.30.0", nil
		},
	})

	require.Empty(t, hint)
}

func TestVersionSkewHintSilentForNonNotFoundErrors(t *testing.T) {
	code := int32(3)
	message := "invalid request"
	err := runAPICallWithResponse(t, http.StatusBadRequest, "application/json",
		mustJSON(t, openapi_client.GooglerpcStatus{Code: &code, Message: &message}))

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.26.0", nil
		},
	})

	require.Empty(t, hint)
}

func TestVersionSkewHintSilentForDevBuilds(t *testing.T) {
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "dev",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.26.0", nil
		},
	})

	require.Empty(t, hint)
}

func TestVersionSkewHintSilentWhenServerVersionCheckFails(t *testing.T) {
	err := notFoundAPIError(t)

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "", context.DeadlineExceeded
		},
	})

	require.Empty(t, hint)
}

// notFoundAPIError reproduces what a CLI newer than the server receives: the
// grpc-gateway on the server answers 404 with a google.rpc.Status body.
func notFoundAPIError(t *testing.T) error {
	t.Helper()

	code := int32(5)
	message := "Not Found"
	return runAPICallWithResponse(t, http.StatusNotFound, "application/json",
		mustJSON(t, openapi_client.GooglerpcStatus{Code: &code, Message: &message}))
}

func TestVersionSkewHintSilentForResourceNotFound(t *testing.T) {
	// A descriptive not-found means the route exists and the resource does
	// not; version skew is not the cause, so no hint.
	code := int32(5)
	message := "canvas not found"
	err := runAPICallWithResponse(t, http.StatusNotFound, "application/json",
		mustJSON(t, openapi_client.GooglerpcStatus{Code: &code, Message: &message}))

	hint := VersionSkewHint(context.Background(), err, BindOptions{
		CLIVersion: "v0.30.0",
		ServerVersion: func(ctx context.Context) (string, error) {
			return "v0.26.0", nil
		},
	})

	require.Empty(t, hint)
}
