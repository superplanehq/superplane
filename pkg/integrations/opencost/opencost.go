package opencost

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("opencost", &OpenCost{})
}

type OpenCost struct{}

func (o *OpenCost) Name() string {
	return "opencost"
}

func (o *OpenCost) Label() string {
	return "OpenCost"
}

func (o *OpenCost) Icon() string {
	return "dollar-sign"
}

func (o *OpenCost) Description() string {
	return "Monitor and query Kubernetes cost allocation data from OpenCost"
}

func (o *OpenCost) Instructions() string {
	return `1. **Deploy OpenCost:** Ensure OpenCost is running in your Kubernetes cluster and its API is accessible.
2. **API URL:** Provide the base URL where the OpenCost API is reachable (e.g. ` + "`http://opencost.opencost.svc:9003`" + `).
3. **No authentication required:** OpenCost does not require authentication by default. If you have placed it behind an authenticating proxy, provide the API token.`
}

func (o *OpenCost) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiURL",
			Label:       "API URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "http://opencost.opencost.svc:9003",
			Description: "Base URL of the OpenCost API",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    false,
			Togglable:   true,
			Description: "Optional bearer token if OpenCost is behind an authenticating proxy",
		},
	}
}

func (o *OpenCost) Components() []core.Component {
	return []core.Component{
		&GetCostAllocation{},
	}
}

func (o *OpenCost) Triggers() []core.Trigger {
	return []core.Trigger{
		&CostExceedsThreshold{},
	}
}

func (o *OpenCost) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify OpenCost connection: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OpenCost) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenCost) Actions() []core.Action {
	return []core.Action{}
}

func (o *OpenCost) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (o *OpenCost) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (o *OpenCost) HandleRequest(ctx core.HTTPRequestContext) {}
