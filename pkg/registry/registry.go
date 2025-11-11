package registry

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/pkg/triggers"
	"gorm.io/gorm"
)

var (
	registeredComponents   = make(map[string]components.Component)
	registeredTriggers     = make(map[string]triggers.Trigger)
	registeredApplications = make(map[string]applications.Application)
	mu                     sync.RWMutex
)

func RegisterComponent(name string, c components.Component) {
	mu.Lock()
	defer mu.Unlock()
	registeredComponents[name] = c
}

func RegisterTrigger(name string, t triggers.Trigger) {
	mu.Lock()
	defer mu.Unlock()
	registeredTriggers[name] = t
}

func RegisterIntegration(name string, i applications.Application) {
	mu.Lock()
	defer mu.Unlock()
	registeredApplications[name] = i
}

type Integration struct {
	EventHandler       integrations.EventHandler
	OIDCVerifier       integrations.OIDCVerifier
	NewResourceManager func(ctx context.Context, URL string, authenticate integrations.AuthenticateFn) (integrations.ResourceManager, error)
}

type Registry struct {
	httpClient   *http.Client
	Encryptor    crypto.Encryptor
	Integrations map[string]Integration
	Applications map[string]applications.Application
	Components   map[string]components.Component
	Triggers     map[string]triggers.Trigger
}

func NewRegistry(encryptor crypto.Encryptor) *Registry {
	r := &Registry{
		Encryptor:    encryptor,
		Integrations: map[string]Integration{},
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		Components:   map[string]components.Component{},
		Triggers:     map[string]triggers.Trigger{},
		Applications: map[string]applications.Application{},
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
	}

	r.Integrations[models.IntegrationTypeGithub] = Integration{
		EventHandler:       &github.GitHubEventHandler{},
		OIDCVerifier:       &github.GitHubOIDCVerifier{},
		NewResourceManager: github.NewGitHubResourceManager,
	}

	//
	// Copy registered integrations, components and triggers
	//
	mu.RLock()
	defer mu.RUnlock()

	for name, component := range registeredComponents {
		r.Components[name] = component
	}

	for name, trigger := range registeredTriggers {
		r.Triggers[name] = trigger
	}

	for name, application := range registeredApplications {
		r.Applications[name] = application
	}
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

func (r *Registry) NewResourceManager(ctx context.Context, integration *models.Integration) (integrations.ResourceManager, error) {
	return r.NewResourceManagerInTransaction(ctx, database.Conn(), integration)
}

func (r *Registry) NewResourceManagerInTransaction(ctx context.Context, tx *gorm.DB, integration *models.Integration) (integrations.ResourceManager, error) {
	registration, ok := r.Integrations[integration.Type]
	if !ok {
		return nil, fmt.Errorf("integration type %s not registered", integration.Type)
	}

	authFn, err := r.getAuthFn(ctx, tx, integration)
	if err != nil {
		return nil, fmt.Errorf("error getting authentication function: %v", err)
	}

	return registration.NewResourceManager(ctx, integration.URL, authFn)
}

func (r *Registry) getAuthFn(ctx context.Context, tx *gorm.DB, integration *models.Integration) (integrations.AuthenticateFn, error) {
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

func (r *Registry) ListTriggers() []triggers.Trigger {
	triggers := make([]triggers.Trigger, 0, len(r.Triggers))
	for _, trigger := range r.Triggers {
		triggers = append(triggers, trigger)
	}

	sort.Slice(triggers, func(i, j int) bool {
		return triggers[i].Name() < triggers[j].Name()
	})

	return triggers
}

func (r *Registry) GetTrigger(name string) (triggers.Trigger, error) {
	trigger, ok := r.Triggers[name]
	if !ok {
		return nil, fmt.Errorf("trigger %s not registered", name)
	}

	return trigger, nil
}

func (r *Registry) ListComponents() []components.Component {
	components := make([]components.Component, 0, len(r.Components))
	for _, component := range r.Components {
		components = append(components, component)
	}

	sort.Slice(components, func(i, j int) bool {
		return components[i].Name() < components[j].Name()
	})

	return components
}

func (r *Registry) GetComponent(name string) (components.Component, error) {
	component, ok := r.Components[name]
	if !ok {
		return nil, fmt.Errorf("component %s not registered", name)
	}

	return component, nil
}

func (r *Registry) GetApplication(name string) (applications.Application, error) {
	application, ok := r.Applications[name]
	if !ok {
		return nil, fmt.Errorf("application %s not registered", name)
	}

	return application, nil
}

func (r *Registry) ListApplications() []applications.Application {
	applications := make([]applications.Application, 0, len(r.Applications))
	for _, application := range r.Applications {
		applications = append(applications, application)
	}

	sort.Slice(applications, func(i, j int) bool {
		return applications[i].Name() < applications[j].Name()
	})

	return applications
}
