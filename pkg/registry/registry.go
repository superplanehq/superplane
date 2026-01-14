package registry

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

var (
	registeredComponents   = make(map[string]core.Component)
	registeredTriggers     = make(map[string]core.Trigger)
	registeredApplications = make(map[string]core.Application)
	registeredWidgets      = make(map[string]core.Widget)
	mu                     sync.RWMutex
)

func RegisterComponent(name string, c core.Component) {
	mu.Lock()
	defer mu.Unlock()
	registeredComponents[name] = c
}

func RegisterTrigger(name string, t core.Trigger) {
	mu.Lock()
	defer mu.Unlock()
	registeredTriggers[name] = t
}

func RegisterApplication(name string, i core.Application) {
	mu.Lock()
	defer mu.Unlock()
	registeredApplications[name] = i
}

func RegisterWidget(name string, w core.Widget) {
	mu.Lock()
	defer mu.Unlock()
	registeredWidgets[name] = w
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
	Applications map[string]core.Application
	Components   map[string]core.Component
	Triggers     map[string]core.Trigger
	Widgets      map[string]core.Widget
}

func NewRegistry(encryptor crypto.Encryptor) *Registry {
	r := &Registry{
		Encryptor:    encryptor,
		Integrations: map[string]Integration{},
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		Components:   map[string]core.Component{},
		Triggers:     map[string]core.Trigger{},
		Applications: map[string]core.Application{},
		Widgets:      map[string]core.Widget{},
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
	// Copy registered components, triggers, and applications with safe wrappers
	//
	mu.RLock()
	defer mu.RUnlock()

	for name, component := range registeredComponents {
		r.Components[name] = NewPanicableComponent(component)
	}

	for name, trigger := range registeredTriggers {
		r.Triggers[name] = NewPanicableTrigger(trigger)
	}

	for name, application := range registeredApplications {
		r.Applications[name] = NewPanicableApplication(application)
	}

	//
	// Widgets are not required to be panicable, since they just carry Configuration data
	// and no logic is executed.
	//
	for name, widget := range registeredWidgets {
		r.Widgets[name] = widget
	}
}

func (r *Registry) GetHTTPClient() *http.Client {
	return r.httpClient
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
		provider, err := secrets.NewProvider(tx, r.Encryptor, secretInfo.Name, integration.DomainType, integration.DomainID)
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

func (r *Registry) ListTriggers() []core.Trigger {
	triggers := make([]core.Trigger, 0, len(r.Triggers))
	for _, trigger := range r.Triggers {
		triggers = append(triggers, trigger)
	}

	sort.Slice(triggers, func(i, j int) bool {
		return triggers[i].Name() < triggers[j].Name()
	})

	return triggers
}

func (r *Registry) GetTrigger(name string) (core.Trigger, error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid trigger name: %s", name)
	}

	if len(parts) == 1 {
		trigger, ok := r.Triggers[name]
		if !ok {
			return nil, fmt.Errorf("trigger %s not registered", name)
		}

		return trigger, nil
	}

	return r.GetApplicationTrigger(parts[0], name)
}

func (r *Registry) ListComponents() []core.Component {
	components := make([]core.Component, 0, len(r.Components))
	for _, component := range r.Components {
		components = append(components, component)
	}

	sort.Slice(components, func(i, j int) bool {
		return components[i].Name() < components[j].Name()
	})

	return components
}

func (r *Registry) GetComponent(name string) (core.Component, error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid component name: %s", name)
	}

	if len(parts) == 1 {
		component, ok := r.Components[name]
		if !ok {
			return nil, fmt.Errorf("component %s not registered", name)
		}

		return component, nil
	}

	return r.GetApplicationComponent(parts[0], name)
}

func (r *Registry) GetWidget(name string) (core.Widget, error) {
	widget, ok := r.Widgets[name]

	if !ok {
		return nil, fmt.Errorf("widget %s not registered", name)
	}

	return widget, nil
}

func (r *Registry) ListWidgets() []core.Widget {
	widgets := make([]core.Widget, 0, len(r.Widgets))
	for _, widget := range r.Widgets {
		widgets = append(widgets, widget)
	}

	sort.Slice(widgets, func(i, j int) bool {
		return widgets[i].Name() < widgets[j].Name()
	})

	return widgets
}

func (r *Registry) GetApplication(name string) (core.Application, error) {
	application, ok := r.Applications[name]
	if !ok {
		return nil, fmt.Errorf("application %s not registered", name)
	}

	return application, nil
}

func (r *Registry) ListApplications() []core.Application {
	applications := make([]core.Application, 0, len(r.Applications))
	for _, application := range r.Applications {
		applications = append(applications, application)
	}

	sort.Slice(applications, func(i, j int) bool {
		return applications[i].Name() < applications[j].Name()
	})

	return applications
}

func (r *Registry) GetApplicationTrigger(appName, triggerName string) (core.Trigger, error) {
	application, err := r.GetApplication(appName)
	if err != nil {
		return nil, err
	}

	for _, trigger := range application.Triggers() {
		if trigger.Name() == triggerName {
			return trigger, nil
		}
	}

	return nil, fmt.Errorf("trigger %s not found for app %s", triggerName, appName)
}

func (r *Registry) GetApplicationComponent(appName, componentName string) (core.Component, error) {
	application, err := r.GetApplication(appName)
	if err != nil {
		return nil, err
	}

	for _, component := range application.Components() {
		if component.Name() == componentName {
			return component, nil
		}
	}

	return nil, fmt.Errorf("component %s not found for app %s", componentName, appName)
}
