package integrations

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
)

const (
	ResourceTypeTask         = "task"
	ResourceTypeTaskTrigger  = "task-trigger"
	ResourceTypeProject      = "project"
	ResourceTypeWorkflow     = "workflow"
	ResourceTypeNotification = "notification"
	ResourceTypeSecret       = "secret"
	ResourceTypePipeline     = "pipeline"
	ResourceTypeRepository   = "repository"
)

type Integration interface {
	Get(resourceType, id string, parentIDs ...string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string, parentIDs ...string) ([]Resource, error)
	HasOidcSupport() bool
}

type Resource interface {
	Id() string
	Name() string
	Type() string
}

func NewIntegration(ctx context.Context, integration *models.Integration, encryptor crypto.Encryptor) (Integration, error) {
	switch integration.Type {
	case models.IntegrationTypeSemaphore:
		secretInfo := integration.Auth.Data().Token.ValueFrom.Secret
		provider, err := secrets.NewProvider(encryptor, secretInfo.Name, integration.DomainID.String())
		if err != nil {
			return nil, fmt.Errorf("error creating secret provider: %v", err)
		}

		values, err := provider.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("error loading values for secret %s: %v", secretInfo.Name, err)
		}

		token, ok := values[secretInfo.Key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in secret %s: %v", secretInfo.Key, secretInfo.Name, err)
		}

		return NewSemaphoreIntegration(integration.URL, token)
	default:
		return nil, fmt.Errorf("unsupported integration type %s", integration.Type)
	}
}
