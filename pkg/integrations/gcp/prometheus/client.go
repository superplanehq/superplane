package prometheus

import (
	"context"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"

	"github.com/superplanehq/superplane/pkg/core"
)

// queryBaseURL is the host+version for Google Cloud Managed Service for
// Prometheus' Prometheus-compatible HTTP API. The per-project query path is
// appended by the URL builders in common.go.
const queryBaseURL = "https://monitoring.googleapis.com/v1"

// Client is the interface used by the Managed Service for Prometheus
// components. Queries are issued against the Prometheus-compatible frontend on
// monitoring.googleapis.com (a different host than Compute), so calls use the
// fully-qualified *URL helper.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	ProjectID() string
}

// newClient is a package variable so tests can inject fake clients.
var newClient = func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	return gcpcommon.NewClient(httpCtx, integration)
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	return newClient(httpCtx, integration)
}
