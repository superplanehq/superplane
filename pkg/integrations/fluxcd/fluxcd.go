package fluxcd

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To connect FluxCD to SuperPlane:

1. **Kubernetes API Server**: Enter the URL of the Kubernetes API server where FluxCD is running (e.g. ` + "`https://kubernetes.example.com:6443`" + `).
2. **Bearer Token**: Create a ServiceAccount with RBAC permissions for Flux resources, then create a token:
   - ` + "`kubectl create serviceaccount superplane -n flux-system`" + `
   - ` + "`kubectl create clusterrolebinding superplane --clusterrole=cluster-admin --serviceaccount=flux-system:superplane`" + `
   - ` + "`kubectl create token superplane -n flux-system`" + `
3. **CA Certificate** (optional): If your cluster uses a custom CA, paste the PEM-encoded certificate.
4. **Namespace**: Default namespace for Flux resources (typically ` + "`flux-system`" + `).

For the **On Reconciliation Completed** trigger:
1. Save the canvas to generate the webhook URL.
2. Configure a FluxCD [Notification Provider](https://fluxcd.io/flux/components/notification/providers/) of type ` + "`generic`" + ` pointing to the generated webhook URL.
3. Create a FluxCD [Alert](https://fluxcd.io/flux/components/notification/alerts/) that references the provider and the resources you want to monitor.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("fluxcd", &FluxCD{}, &FluxCDWebhookHandler{})
}

type FluxCD struct{}

type Configuration struct {
	Server        string `json:"server" mapstructure:"server"`
	Token         string `json:"token" mapstructure:"token"`
	CACertificate string `json:"caCertificate" mapstructure:"caCertificate"`
	Namespace     string `json:"namespace" mapstructure:"namespace"`
}

func (f *FluxCD) Name() string {
	return "fluxcd"
}

func (f *FluxCD) Label() string {
	return "Flux CD"
}

func (f *FluxCD) Icon() string {
	return "git-branch"
}

func (f *FluxCD) Description() string {
	return "Build GitOps workflows with Flux CD reconciliation triggers and actions"
}

func (f *FluxCD) Instructions() string {
	return installationInstructions
}

func (f *FluxCD) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "server",
			Label:       "Kubernetes API Server",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "URL of the Kubernetes API server (e.g. https://kubernetes.example.com:6443)",
			Placeholder: "https://kubernetes.example.com:6443",
		},
		{
			Name:        "token",
			Label:       "Bearer Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Kubernetes ServiceAccount bearer token with access to Flux resources",
		},
		{
			Name:        "caCertificate",
			Label:       "CA Certificate",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "PEM-encoded CA certificate for the Kubernetes API server (optional)",
			Placeholder: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
		},
		{
			Name:        "namespace",
			Label:       "Default Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "flux-system",
			Description: "Default namespace for Flux resources",
			Placeholder: "flux-system",
		},
	}
}

func (f *FluxCD) Components() []core.Component {
	return []core.Component{
		&ReconcileSource{},
	}
}

func (f *FluxCD) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnReconciliationCompleted{},
	}
}

func (f *FluxCD) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (f *FluxCD) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.Server == "" {
		return fmt.Errorf("server is required")
	}

	if config.Token == "" {
		return fmt.Errorf("token is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes API: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (f *FluxCD) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
}

func (f *FluxCD) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (f *FluxCD) Actions() []core.Action {
	return []core.Action{}
}

func (f *FluxCD) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
