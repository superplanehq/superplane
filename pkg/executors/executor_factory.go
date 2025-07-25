package executors

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

type BuildFn func(integrations.Integration, integrations.Resource) (Executor, error)

var executorTypes = map[string]BuildFn{}

func Register(name string, builder BuildFn) {
	executorTypes[name] = builder
}

func NewExecutor(executor *models.StageExecutor, encryptor crypto.Encryptor) (Executor, error) {
	builder, ok := executorTypes[executor.Type]
	if !ok {
		return nil, fmt.Errorf("executor type %s not registered", executor.Type)
	}

	//
	// Executor does not require integration
	//
	if executor.ResourceID == nil {
		return builder(nil, nil)
	}

	//
	// Executor requires integration,
	// so we need to instantiate a new integration for it.
	//
	r, err := executor.FindIntegration()
	if err != nil {
		return nil, fmt.Errorf("error finding integration: %v", err)
	}

	integration, err := integrations.NewIntegration(context.Background(), r, encryptor)
	if err != nil {
		return nil, fmt.Errorf("error creating integration: %v", err)
	}

	resource, err := executor.GetResource()
	if err != nil {
		return nil, fmt.Errorf("error getting resource: %v", err)
	}

	return builder(integration, resource)
}
