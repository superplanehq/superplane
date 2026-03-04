package cloudfunctions

import (
	"context"
	"fmt"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

const cloudFunctionsBaseURL = "https://cloudfunctions.googleapis.com"

// Client is the interface used by Cloud Functions components to call the API.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
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
		panic("gcp cloudfunctions: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}

// functionCallURL returns the Cloud Functions v1 API URL for calling a function.
// It works with both v1 and v2 resource names, which share the same format:
// projects/{project}/locations/{location}/functions/{function}
func functionCallURL(resourceName string) string {
	return fmt.Sprintf("%s/v1/%s:call", cloudFunctionsBaseURL, resourceName)
}

// functionGetURL returns the Cloud Functions v2 API URL for getting function details.
func functionGetURL(resourceName string) string {
	return fmt.Sprintf("%s/v2/%s", cloudFunctionsBaseURL, resourceName)
}
