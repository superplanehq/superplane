package compute

import (
	"context"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/core"
	"sync"
)

type Client interface {
	CreateInstance(ctx context.Context, request core.LaunchInstanceRequest) (core.LaunchInstanceResponse, error)
	GetInstance(ctx context.Context, request core.GetInstanceRequest) (core.GetInstanceResponse, error)
	UpdateInstance(ctx context.Context, request core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error)
	TerminateInstance(ctx context.Context, request core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error)
	InstanceAction(ctx context.Context, request core.InstanceActionRequest) (core.InstanceActionResponse, error)
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
		panic("oci compute: SetClientFactory was not called by the oci integration")
	}
	return fn(ctx)
}
