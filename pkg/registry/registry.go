package registry

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

type Registry struct {
	IntegrationRegistry *IntegrationRegistry
	ExecutorRegistry    *ExecutorRegistry
}

func NewRegistry(encryptor crypto.Encryptor) *Registry {
	r := &Registry{
		IntegrationRegistry: NewIntegrationRegistry(encryptor),
		ExecutorRegistry:    NewExecutorRegistry(encryptor),
	}

	r.IntegrationRegistry.Init()
	r.ExecutorRegistry.Init()

	return r
}

func (r *Registry) HasIntegrationWithType(integrationType string) bool {
	_, ok := r.IntegrationRegistry.Integrations[integrationType]
	return ok
}

func (r *Registry) NewIntegration(ctx context.Context, integration *models.Integration) (integrations.Integration, error) {
	return r.IntegrationRegistry.New(ctx, integration)
}

func (r *Registry) NewExecutor(executorType string, integration *models.Integration, resource integrations.Resource) (executors.Executor, error) {
	builder, ok := r.ExecutorRegistry.Executors[executorType]
	if !ok {
		return nil, fmt.Errorf("executor type %s not registered", executorType)
	}

	//
	// Executor does not require integration
	//
	if integration == nil {
		return builder(nil, nil)
	}

	//
	// Executor requires integration,
	// so we need to instantiate a new integration for it.
	//
	integrationImpl, err := r.IntegrationRegistry.New(context.Background(), integration)
	if err != nil {
		return nil, fmt.Errorf("error creating integration: %v", err)
	}

	return builder(integrationImpl, resource)
}
