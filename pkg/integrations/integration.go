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
		provider, err := secretProvider(encryptor, secretInfo, integration)
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

func secretProvider(encryptor crypto.Encryptor, secretDef *models.ValueDefinitionFromSecret, integration *models.Integration) (secrets.Provider, error) {
	//
	// If the integration is scoped to an organization, the secret must also be scoped there.
	//
	if integration.DomainType == models.DomainTypeOrganization {
		return secrets.NewProvider(encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Here, we know the integration is on the canvas level.
	// If the secret is also on the canvas level, we use the same domain type and ID.
	//
	if secretDef.DomainType == models.DomainTypeCanvas {
		return secrets.NewProvider(encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Otherwise, the integration is on the canvas level, but the secret is on the organization level,
	// so we need to get the organization ID for the canvas where the integration is.
	//
	canvas, err := models.FindCanvasByID(integration.DomainID.String())
	if err != nil {
		return nil, fmt.Errorf("error finding canvas %s: %v", integration.DomainID, err)
	}

	return secrets.NewProvider(encryptor, secretDef.Name, secretDef.DomainType, canvas.OrganizationID)
}
