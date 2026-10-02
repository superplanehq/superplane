package monitoring

import (
	"context"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"

	"github.com/superplanehq/superplane/pkg/core"
)

const monitoringBaseURL = "https://monitoring.googleapis.com/v3"

// Client is the interface used by Cloud Monitoring components to call the API.
// Alert policies live on monitoring.googleapis.com (a different host than
// Compute), so every call uses the fully-qualified *URL helpers.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	PatchURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	DeleteURL(ctx context.Context, fullURL string) ([]byte, error)
	ProjectID() string
}

// newClient is a package variable so tests can inject fake clients.
var newClient = func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	return gcpcommon.NewClient(httpCtx, integration)
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	return newClient(httpCtx, integration)
}
