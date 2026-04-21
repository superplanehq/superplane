package compute

import (
	"context"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	spcore "github.com/superplanehq/superplane/pkg/core"
	"sync"
)

type Client interface {
	LaunchInstance(ctx context.Context, request ocicore.LaunchInstanceRequest) (ocicore.LaunchInstanceResponse, error)
	GetInstance(ctx context.Context, request ocicore.GetInstanceRequest) (ocicore.GetInstanceResponse, error)
	UpdateInstance(ctx context.Context, request ocicore.UpdateInstanceRequest) (ocicore.UpdateInstanceResponse, error)
	TerminateInstance(ctx context.Context, request ocicore.TerminateInstanceRequest) (ocicore.TerminateInstanceResponse, error)
	InstanceAction(ctx context.Context, request ocicore.InstanceActionRequest) (ocicore.InstanceActionResponse, error)
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(ctx spcore.ExecutionContext) (Client, error)
)

func SetClientFactory(fn func(ctx spcore.ExecutionContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(ctx spcore.ExecutionContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("oci compute: SetClientFactory was not called by the oci integration")
	}
	return fn(ctx)
}
