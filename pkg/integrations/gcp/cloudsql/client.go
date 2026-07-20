package cloudsql

import (
	"context"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

// sqlAdminBaseURL is the host+version for the Cloud SQL Admin API. Cloud SQL is
// hosted on sqladmin.googleapis.com (a different host than Compute), so every
// call uses the fully-qualified *URL helpers.
const sqlAdminBaseURL = "https://sqladmin.googleapis.com/v1"

// Client is the interface used by the Cloud SQL components.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	DeleteURL(ctx context.Context, fullURL string) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)
)

func SetClientFactory(fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp cloudsql: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}
