package registry

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
)

type IntegrationRegistry struct {
	Integrations map[string]integrations.BuildFn
	Encryptor    crypto.Encryptor
}

func NewIntegrationRegistry(encryptor crypto.Encryptor) *IntegrationRegistry {
	r := &IntegrationRegistry{
		Encryptor:    encryptor,
		Integrations: map[string]integrations.BuildFn{},
	}

	r.Init()
	return r
}

func (r *IntegrationRegistry) Init() {
	r.Integrations[models.IntegrationTypeSemaphore] = semaphore.NewSemaphoreIntegration
}

func (r *IntegrationRegistry) New(ctx context.Context, integration *models.Integration) (integrations.Integration, error) {
	builder, ok := r.Integrations[integration.Type]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integration.Type)
	}

	authFn, err := r.getAuthFn(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("error getting authentication function: %v", err)
	}

	return builder(ctx, integration, authFn)
}

func (r *IntegrationRegistry) getAuthFn(ctx context.Context, integration *models.Integration) (integrations.AuthenticateFn, error) {
	switch integration.AuthType {
	case models.IntegrationAuthTypeToken:
		secretInfo := integration.Auth.Data().Token.ValueFrom.Secret
		provider, err := r.secretProvider(secretInfo, integration)
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

		return func() (string, error) {
			return token, nil
		}, nil
	}

	return nil, fmt.Errorf("integration auth type %s not supported", integration.AuthType)
}

func (r *IntegrationRegistry) secretProvider(secretDef *models.ValueDefinitionFromSecret, integration *models.Integration) (secrets.Provider, error) {
	//
	// If the integration is scoped to an organization, the secret must also be scoped there.
	//
	if integration.DomainType == models.DomainTypeOrganization {
		return secrets.NewProvider(r.Encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Here, we know the integration is on the canvas level.
	// If the secret is also on the canvas level, we use the same domain type and ID.
	//
	if secretDef.DomainType == models.DomainTypeCanvas {
		return secrets.NewProvider(r.Encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Otherwise, the integration is on the canvas level, but the secret is on the organization level,
	// so we need to get the organization ID for the canvas where the integration is.
	//
	canvas, err := models.FindCanvasByID(integration.DomainID.String())
	if err != nil {
		return nil, fmt.Errorf("error finding canvas %s: %v", integration.DomainID, err)
	}

	return secrets.NewProvider(r.Encryptor, secretDef.Name, secretDef.DomainType, canvas.OrganizationID)
}
