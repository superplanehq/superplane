package createvm

import (
	"context"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Post(ctx context.Context, path string, body any) ([]byte, error)
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(ctx core.ExecutionContext) (Client, error)
)

func SetClientFactory(fn func(ctx core.ExecutionContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(ctx core.ExecutionContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp createvm: SetClientFactory was not called by the gcp integration")
	}
	return fn(ctx)
}
