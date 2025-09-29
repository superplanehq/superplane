package registry

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/executors/http"
	"github.com/superplanehq/superplane/pkg/executors/noop"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

type Integration struct {
	EventHandler       integrations.EventHandler
	OIDCVerifier       integrations.OIDCVerifier
	NewResourceManager func(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.ResourceManager, error)
	NewExecutor        func(integrations.ResourceManager, integrations.Resource) (integrations.Executor, error)
}

type Registry struct {
	Encryptor    crypto.Encryptor
	Integrations map[string]Integration
	Executors    map[string]executors.Executor
}

func NewRegistry(encryptor crypto.Encryptor) *Registry {
	r := &Registry{
		Encryptor:    encryptor,
		Executors:    map[string]executors.Executor{},
		Integrations: map[string]Integration{},
	}

	r.Init()

	return r
}

func (r *Registry) Init() {
	//
	// Register the integrations
	//
	r.Integrations[models.IntegrationTypeSemaphore] = Integration{
		EventHandler:       &semaphore.SemaphoreEventHandler{},
		OIDCVerifier:       &semaphore.SemaphoreOIDCVerifier{},
		NewResourceManager: semaphore.NewSemaphoreResourceManager,
		NewExecutor:        semaphore.NewSemaphoreExecutor,
	}

	r.Integrations[models.IntegrationTypeGithub] = Integration{
		EventHandler:       &github.GitHubEventHandler{},
		OIDCVerifier:       &github.GitHubOIDCVerifier{},
		NewResourceManager: github.NewGitHubResourceManager,
		NewExecutor:        github.NewGitHubExecutor,
	}

	//
	// Register the executors
	//
	r.Executors[models.ExecutorTypeHTTP] = http.NewHTTPExecutor()
	r.Executors[models.ExecutorTypeNoOp] = noop.NewNoOpExecutor()
}

func (r *Registry) HasIntegrationWithType(integrationType string) bool {
	_, ok := r.Integrations[integrationType]
	return ok
}

func (r *Registry) GetEventHandler(integrationType string) (integrations.EventHandler, error) {
	registration, ok := r.Integrations[integrationType]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integrationType)
	}

	return registration.EventHandler, nil
}

func (r *Registry) HasOIDCVerifier(integrationType string) bool {
	integration, ok := r.Integrations[integrationType]
	if !ok {
		return false
	}

	return integration.OIDCVerifier != nil
}

func (r *Registry) GetOIDCVerifier(integrationType string) (integrations.OIDCVerifier, error) {
	registration, ok := r.Integrations[integrationType]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integrationType)
	}

	if registration.OIDCVerifier == nil {
		return nil, fmt.Errorf("integration type %s does not support OIDC verification", integrationType)
	}

	return registration.OIDCVerifier, nil
}

func (r *Registry) NewResourceManager(tx *gorm.DB, ctx context.Context, integration *models.Integration) (integrations.ResourceManager, error) {
	registration, ok := r.Integrations[integration.Type]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integration.Type)
	}

	authFn, err := r.getAuthFn(tx, ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("error getting authentication function: %v", err)
	}

	return registration.NewResourceManager(ctx, integration.URL, authFn)
}

func (r *Registry) getAuthFn(tx *gorm.DB, ctx context.Context, integration *models.Integration) (integrations.AuthenticateFn, error) {
	switch integration.AuthType {
	case models.IntegrationAuthTypeToken:
		secretInfo := integration.Auth.Data().Token.ValueFrom.Secret
		provider, err := r.secretProvider(tx, secretInfo, integration)
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

func (r *Registry) secretProvider(tx *gorm.DB, secretDef *models.ValueDefinitionFromSecret, integration *models.Integration) (secrets.Provider, error) {
	//
	// If the integration is scoped to an organization, the secret must also be scoped there.
	//
	if integration.DomainType == models.DomainTypeOrganization {
		return secrets.NewProvider(tx, r.Encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Here, we know the integration is on the canvas level.
	// If the secret is also on the canvas level, we use the same domain type and ID.
	//
	if secretDef.DomainType == models.DomainTypeCanvas {
		return secrets.NewProvider(tx, r.Encryptor, secretDef.Name, secretDef.DomainType, integration.DomainID)
	}

	//
	// Otherwise, the integration is on the canvas level, but the secret is on the organization level,
	// so we need to get the organization ID for the canvas where the integration is.
	//
	canvas, err := models.FindUnscopedCanvasByID(integration.DomainID.String())
	if err != nil {
		return nil, fmt.Errorf("error finding canvas %s: %v", integration.DomainID, err)
	}

	return secrets.NewProvider(tx, r.Encryptor, secretDef.Name, secretDef.DomainType, canvas.OrganizationID)
}

func (r *Registry) NewIntegrationExecutor(tx *gorm.DB, integration *models.Integration, resource integrations.Resource) (integrations.Executor, error) {
	if integration == nil {
		return nil, fmt.Errorf("integration is required")
	}

	resourceManager, err := r.NewResourceManager(tx, context.Background(), integration)
	if err != nil {
		return nil, fmt.Errorf("error creating integration: %v", err)
	}

	registration, ok := r.Integrations[integration.Type]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integration.Type)
	}

	return registration.NewExecutor(resourceManager, resource)
}

func (r *Registry) NewExecutor(executorType string) (executors.Executor, error) {
	executor, ok := r.Executors[executorType]
	if !ok {
		return nil, fmt.Errorf("executor type %s not registered", executorType)
	}

	return executor, nil
}
