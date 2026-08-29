package argocd

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("argocd", &ArgoCD{})
}

type ArgoCD struct{}

type Configuration struct {
	ServerURL string `json:"serverUrl" mapstructure:"serverUrl"`
	AuthToken string `json:"authToken" mapstructure:"authToken"`
}

func (a *ArgoCD) Name() string {
	return "argocd"
}

func (a *ArgoCD) Label() string {
	return "Argo CD"
}

func (a *ArgoCD) Icon() string {
	return "kubernetes"
}

func (a *ArgoCD) Description() string {
	return "Connect to Argo CD to manage and observe GitOps application delivery"
}

func (a *ArgoCD) Instructions() string {
	return `1. **Server URL:** Enter the external URL for your Argo CD server.
2. **Authentication token:** Create a project role token with ` + "`applications, get`" + ` permission.
3. **Additional permissions:** Add only the permissions required by the components you use.
4. **Connect:** Paste the token in SuperPlane and connect the integration.`
}

func (a *ArgoCD) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serverUrl",
			Label:       "Server URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://argocd.example.com",
			Description: "External URL for the Argo CD server",
		},
		{
			Name:        "authToken",
			Label:       "Authentication Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Argo CD bearer token with the required application permissions",
		},
	}
}

func (a *ArgoCD) Actions() []core.Action {
	return nil
}

func (a *ArgoCD) Triggers() []core.Trigger {
	return nil
}

func (a *ArgoCD) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Argo CD credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (a *ArgoCD) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (a *ArgoCD) HandleRequest(ctx core.HTTPRequestContext) {}

func (a *ArgoCD) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (a *ArgoCD) Hooks() []core.Hook {
	return []core.Hook{}
}

func (a *ArgoCD) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
