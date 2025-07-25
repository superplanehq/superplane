package integrations

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
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

	FeatureOpenIdConnectToken = "oidc"
)

type Integration interface {
	Get(resourceType, id string, parentIDs ...string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string, parentIDs ...string) ([]Resource, error)
	HasSupportFor(string) bool
	ValidateOpenIDConnectClaims(idToken *oidc.IDToken, resourceId, parentId string) error
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
		provider, err := secrets.NewProvider(encryptor, secretInfo.Name, integration.DomainType, integration.DomainID)
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

// TODO: I don't like this.
// We should come up with a better way to instantiate integrations
// without requiring a switch statement like this.
func NewIntegrationWithoutAuth(ctx context.Context, integrationType string) (Integration, error) {
	switch integrationType {
	case models.IntegrationTypeSemaphore:
		return NewSemaphoreIntegration("", "")
	default:
		return nil, fmt.Errorf("unsupported integration type %s", integrationType)
	}
}
