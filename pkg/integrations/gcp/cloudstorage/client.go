package cloudstorage

import (
	"context"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	storageBaseURL = "https://storage.googleapis.com/storage/v1"
	uploadBaseURL  = "https://storage.googleapis.com/upload/storage/v1"
)

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
		panic("gcp cloudstorage: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}
