package harness

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("harness", &Harness{}, &HarnessWebhookHandler{})
}

type Harness struct{}

func (h *Harness) Name() string {
	return "harness"
}

func (h *Harness) Label() string {
	return "Harness"
}

func (h *Harness) Icon() string {
	return "workflow"
}

func (h *Harness) Description() string {
	return "Run and monitor Harness pipelines from SuperPlane workflows"
}

func (h *Harness) Instructions() string {
	return `1. **Create API key:** In Harness, create a service-account API key with permission to run and read pipeline executions.
2. **Connect once, then configure nodes:** Scope fields (**Org**, **Project**, **Pipeline**) are selected in each Harness node.
3. **Account ID is automatic:** SuperPlane resolves account scope from your API key.
4. **Trigger notifications are automatic:** For **On Pipeline Completed** with a selected **Pipeline**, SuperPlane provisions a pipeline notification rule for you.
5. **Auth method:** SuperPlane calls Harness APIs with ` + "`x-api-key: <token>`" + ` against ` + "`https://app.harness.io/gateway`" + ` unless overridden by Base URL.`
}

func (h *Harness) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Harness API key used for API calls",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Default:     DefaultBaseURL,
			Placeholder: "https://app.harness.io/gateway",
			Description: "Override only for custom or self-hosted Harness gateways",
		},
	}
}

func (h *Harness) Components() []core.Component {
	return []core.Component{
		&RunPipeline{},
	}
}

func (h *Harness) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineCompleted{},
	}
}

func (h *Harness) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Harness) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Harness credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (h *Harness) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (h *Harness) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != ResourceTypeOrg &&
		resourceType != ResourceTypeProject &&
		resourceType != ResourceTypePipeline {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case ResourceTypeOrg:
		organizations, err := client.ListOrganizations()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(organizations))
		for _, organization := range organizations {
			if strings.TrimSpace(organization.Identifier) == "" {
				continue
			}
			name := firstNonEmpty(strings.TrimSpace(organization.Name), organization.Identifier)
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeOrg,
				Name: name,
				ID:   organization.Identifier,
			})
		}
		return resources, nil
	case ResourceTypeProject:
		orgID := strings.TrimSpace(ctx.Parameters["orgId"])
		if orgID == "" {
			return []core.IntegrationResource{}, nil
		}
		projects, err := client.ListProjects(orgID)
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(projects))
		for _, project := range projects {
			if strings.TrimSpace(project.Identifier) == "" {
				continue
			}
			name := firstNonEmpty(strings.TrimSpace(project.Name), project.Identifier)
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeProject,
				Name: name,
				ID:   project.Identifier,
			})
		}
		return resources, nil
	case ResourceTypePipeline:
		orgID := strings.TrimSpace(ctx.Parameters["orgId"])
		projectID := strings.TrimSpace(ctx.Parameters["projectId"])
		if orgID == "" || projectID == "" {
			return []core.IntegrationResource{}, nil
		}

		scopedClient := client.withScope(orgID, projectID)
		pipelines, err := scopedClient.ListPipelines()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(pipelines))
		for _, pipeline := range pipelines {
			if strings.TrimSpace(pipeline.Identifier) == "" {
				continue
			}

			name := strings.TrimSpace(pipeline.Name)
			if name == "" {
				name = pipeline.Identifier
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypePipeline,
				Name: name,
				ID:   pipeline.Identifier,
			})
		}
		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (h *Harness) Actions() []core.Action {
	return []core.Action{}
}

func (h *Harness) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
