package restate

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Restate to work with SuperPlane:

1. **Admin URL**: The URL of your Restate Admin API (default: http://localhost:9070)
2. **Ingress URL**: The URL of your Restate Ingress API (default: http://localhost:8080)
3. **Auth Token** (optional): A Bearer token for authenticating requests to your Restate server

Both URLs must be reachable from the SuperPlane instance. For Restate Cloud,
use the endpoints provided in your Restate Cloud dashboard.
`

func init() {
	registry.RegisterIntegration("restate", &Restate{})
}

type Restate struct{}

type Configuration struct {
	AdminURL   string `json:"adminUrl" mapstructure:"adminUrl"`
	IngressURL string `json:"ingressUrl" mapstructure:"ingressUrl"`
	AuthToken  string `json:"authToken" mapstructure:"authToken"`
}

func (r *Restate) Name() string {
	return "restate"
}

func (r *Restate) Label() string {
	return "Restate"
}

func (r *Restate) Icon() string {
	return "repeat"
}

func (r *Restate) Description() string {
	return "Invoke handlers, manage deployments, and control invocations in Restate"
}

func (r *Restate) Instructions() string {
	return installationInstructions
}

func (r *Restate) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "adminUrl",
			Label:       "Admin URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Restate Admin API URL (e.g. http://localhost:9070)",
			Placeholder: "http://localhost:9070",
		},
		{
			Name:        "ingressUrl",
			Label:       "Ingress URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Restate Ingress API URL (e.g. http://localhost:8080)",
			Placeholder: "http://localhost:8080",
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Optional Bearer token for authenticating requests",
		},
	}
}

func (r *Restate) Actions() []core.Action {
	return []core.Action{
		&InvokeHandler{},
		&SendHandler{},
		&SendDelayedHandler{},
		&RegisterDeployment{},
		&RemoveDeployment{},
		&GetService{},
		&ListServices{},
		&CancelInvocation{},
		&KillInvocation{},
		&PurgeInvocation{},
		&HealthCheck{},
	}
}

func (r *Restate) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (r *Restate) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Restate) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.AdminURL == "" {
		return fmt.Errorf("adminUrl is required")
	}

	if config.IngressURL == "" {
		return fmt.Errorf("ingressUrl is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.CheckHealth()
	if err != nil {
		return fmt.Errorf("failed to connect to Restate: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (r *Restate) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (r *Restate) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (r *Restate) Hooks() []core.Hook {
	return []core.Hook{}
}

func (r *Restate) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
