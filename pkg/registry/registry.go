package registry

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

var (
	registeredComponents   = make(map[string]core.Component)
	registeredTriggers     = make(map[string]core.Trigger)
	registeredIntegrations = make(map[string]core.Integration)
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

func RegisterIntegration(name string, i core.Integration) {
	mu.Lock()
	defer mu.Unlock()
	registeredIntegrations[name] = i
}

func RegisterWidget(name string, w core.Widget) {
	mu.Lock()
	defer mu.Unlock()
	registeredWidgets[name] = w
}

type Registry struct {
	httpClient   *http.Client
	Encryptor    crypto.Encryptor
	Integrations map[string]core.Integration
	Components   map[string]core.Component
	Triggers     map[string]core.Trigger
	Widgets      map[string]core.Widget
}

func NewRegistry(encryptor crypto.Encryptor) *Registry {
	r := &Registry{
		Encryptor:    encryptor,
		httpClient:   NewSSRFSafeHTTPClient(30 * time.Second),
		Components:   map[string]core.Component{},
		Triggers:     map[string]core.Trigger{},
		Integrations: map[string]core.Integration{},
		Widgets:      map[string]core.Widget{},
	}

	r.Init()

	return r
}

func (r *Registry) Init() {
	//
	// Copy registered components, triggers, and integrations with safe wrappers
	//
	mu.RLock()
	defer mu.RUnlock()

	for name, component := range registeredComponents {
		r.Components[name] = NewPanicableComponent(component)
	}

	for name, trigger := range registeredTriggers {
		r.Triggers[name] = NewPanicableTrigger(trigger)
	}

	for name, integration := range registeredIntegrations {
		r.Integrations[name] = NewPanicableIntegration(integration)
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

	return r.GetIntegrationTrigger(parts[0], name)
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

	return r.GetIntegrationComponent(parts[0], name)
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

func (r *Registry) GetIntegration(name string) (core.Integration, error) {
	integration, ok := r.Integrations[name]
	if !ok {
		return nil, fmt.Errorf("integration %s not registered", name)
	}

	return integration, nil
}

func (r *Registry) ListIntegrations() []core.Integration {
	integrations := make([]core.Integration, 0, len(r.Integrations))
	for _, integration := range r.Integrations {
		integrations = append(integrations, integration)
	}

	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].Name() < integrations[j].Name()
	})

	return integrations
}

func (r *Registry) GetIntegrationTrigger(appName, triggerName string) (core.Trigger, error) {
	integration, err := r.GetIntegration(appName)
	if err != nil {
		return nil, err
	}

	for _, trigger := range integration.Triggers() {
		if trigger.Name() == triggerName {
			return trigger, nil
		}
	}

	return nil, fmt.Errorf("trigger %s not found for integration %s", triggerName, appName)
}

func (r *Registry) GetIntegrationComponent(appName, componentName string) (core.Component, error) {
	integration, err := r.GetIntegration(appName)
	if err != nil {
		return nil, err
	}

	for _, component := range integration.Components() {
		if component.Name() == componentName {
			return component, nil
		}
	}

	return nil, fmt.Errorf("component %s not found for integration %s", componentName, appName)
}
