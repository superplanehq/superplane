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

func NewExecutor(executorType string, integration *models.Integration, resource integrations.Resource, encryptor crypto.Encryptor) (Executor, error) {
	builder, ok := executorTypes[executorType]
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
	integrationImpl, err := integrations.NewIntegration(context.Background(), integration, encryptor)
	if err != nil {
		return nil, fmt.Errorf("error creating integration: %v", err)
	}

	return builder(integrationImpl, resource)
}
