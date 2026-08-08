package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
	"google.golang.org/grpc/codes"
)

// ErrServerVersionUnavailable means the server answered but does not expose
// the version endpoint, i.e. it predates version reporting.
var ErrServerVersionUnavailable = errors.New("server does not report its version")

const serverVersionTimeout = 3 * time.Second

// VersionSkewHint explains "not found" API errors that are typically caused
// by version skew on self-hosted installations: the CLI calls an endpoint
// that does not exist on the (older) server. It returns an empty string when
// the error is not a not-found API error, when versions match, or when the
// server version cannot be determined reliably.
func VersionSkewHint(ctx context.Context, err error, options BindOptions) string {
	if err == nil || options.CLIVersion == "" || options.CLIVersion == "dev" || options.ServerVersion == nil {
		return ""
	}

	if !isRouteNotFoundAPIError(err) {
		return ""
	}

	checkCtx, cancel := context.WithTimeout(ctx, serverVersionTimeout)
	defer cancel()

	serverVersion, versionErr := options.ServerVersion(checkCtx)
	switch {
	case errors.Is(versionErr, ErrServerVersionUnavailable):
		return fmt.Sprintf(
			"hint: this CLI is %s but the server does not report its version, so it likely predates this command. Use a CLI release matching your server, or upgrade the server.",
			options.CLIVersion,
		)
	case versionErr != nil:
		return ""
	case serverVersion != "dev" && IsNewerVersion(serverVersion, options.CLIVersion):
		return fmt.Sprintf(
			"hint: CLI version %s is newer than server version %s, so this command's API may not exist on the server yet. Use a matching CLI release, or upgrade the server.",
			options.CLIVersion, serverVersion,
		)
	default:
		return ""
	}
}

// isRouteNotFoundAPIError reports whether the error looks like a missing
// ROUTE rather than a missing resource. The grpc-gateway answers unknown
// paths with the generic "Not Found" message, while handlers that miss a
// resource return descriptive messages such as "canvas not found"; only the
// former suggests version skew.
func isRouteNotFoundAPIError(err error) bool {
	var apiErr *openapi_client.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return false
	}

	if status := extractGoogleRPCStatus(apiErr); status != nil {
		return codes.Code(status.GetCode()) == codes.NotFound &&
			strings.EqualFold(strings.TrimSpace(status.GetMessage()), "Not Found")
	}

	return strings.Contains(apiErr.Error(), "404")
}
