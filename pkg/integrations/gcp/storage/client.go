package storage

import (
	"context"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"

	"github.com/superplanehq/superplane/pkg/core"
)

// storageBaseURL is the host+version for the Cloud Storage JSON API. Cloud
// Storage is hosted on storage.googleapis.com (a different host than Compute),
// so every call uses the fully-qualified *URL helpers.
const storageBaseURL = "https://storage.googleapis.com/storage/v1"

// Client is the interface used by the Cloud Storage components.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
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
