package render

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("render", &Render{}, &RenderWebhookHandler{})
}

type Render struct{}

type Configuration struct {
	APIKey        string `json:"apiKey" mapstructure:"apiKey"`
	Workspace     string `json:"workspace" mapstructure:"workspace"`
	WorkspacePlan string `json:"workspacePlan" mapstructure:"workspacePlan"`
}

type Metadata struct {
	Workspace *WorkspaceMetadata `json:"workspace,omitempty" mapstructure:"workspace"`
}

type WorkspaceMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Plan string `json:"plan" mapstructure:"plan"`
}

func (r *Render) Name() string {
	return "render"
}

func (r *Render) Label() string {
	return "Render"
}

func (r *Render) Icon() string {
	return "server"
}

func (r *Render) Description() string {
	return "Deploy and manage Render services, and react to Render deploy/build events"
}

func (r *Render) Instructions() string {
	return `
1. **API Key:** Create it in [Render Account Settings -> API Keys](https://dashboard.render.com/u/settings#api-keys).
2. **Workspace (optional):** Use your Render workspace ID (` + "`usr-...`" + ` or ` + "`tea-...`" + `) or workspace name. Leave empty to use the first workspace available to the API key.
3. **Workspace Plan:** Select **Professional** or **Organization / Enterprise** (used to choose webhook strategy).
4. **Auth:** SuperPlane sends requests to [Render API v1](https://api.render.com/v1/) using ` + "`Authorization: Bearer <API_KEY>`" + `.
5. **Webhooks:** SuperPlane configures Render webhooks automatically via the [Render Webhooks API](https://render.com/docs/webhooks). No manual setup is required.
6. **Troubleshooting:** Check [Render Dashboard -> Integrations -> Webhooks](https://dashboard.render.com/) and the [Render webhook docs](https://render.com/docs/webhooks).

Note: **Plan requirement:** Render webhooks require a Professional plan or higher.`
}

func (r *Render) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Render API key",
		},
		{
			Name:        "workspace",
			Label:       "Workspace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Render workspace ID/name. Use this if your API key has access to multiple workspaces.",
		},
		{
			Name:     "workspacePlan",
			Label:    "Workspace Plan",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  workspacePlanProfessional,
			Description: "Render workspace plan used for webhook strategy. " +
				"Use Organization / Enterprise when available.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Professional", Value: workspacePlanProfessional},
						{Label: "Organization / Enterprise", Value: workspacePlanOrganization},
					},
				},
			},
		},
	}
}

func (r *Render) Components() []core.Component {
	return []core.Component{
		&Deploy{},
		&GetService{},
		&GetDeploy{},
		&CancelDeploy{},
		&RollbackDeploy{},
		&PurgeCache{},
		&UpdateEnvVar{},
	}
}

func (r *Render) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeploy{},
		&OnBuild{},
	}
}

func (r *Render) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Render) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Render credentials: %w", err)
	}

	workspace, err := resolveWorkspace(client, config.Workspace)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	ctx.Integration.SetMetadata(buildMetadata(workspace.ID, config.WorkspacePlan))
	ctx.Integration.Ready()
	return nil
}

func (r *Render) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (r *Render) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "service" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	workspaceID, err := workspaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	services, err := client.ListServices(workspaceID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(services))
	for _, service := range services {
		if service.ID == "" || service.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: resourceType, Name: service.Name, ID: service.ID})
	}

	return resources, nil
}

func (r *Render) Actions() []core.Action {
	return []core.Action{}
}

func (r *Render) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func workspaceIDForIntegration(client *Client, integration core.IntegrationContext) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err == nil && metadata.Workspace != nil && metadata.Workspace.ID != "" {
		return metadata.Workspace.ID, nil
	}

	workspace := ""
	workspaceValue, workspaceErr := integration.GetConfig("workspace")
	if workspaceErr == nil {
		workspace = string(workspaceValue)
	}

	selectedWorkspace, err := resolveWorkspace(client, workspace)
	if err != nil {
		return "", err
	}

	workspacePlan := workspacePlanProfessional
	workspacePlanValue, workspacePlanErr := integration.GetConfig("workspacePlan")
	if workspacePlanErr == nil {
		workspacePlan = string(workspacePlanValue)
	}

	integration.SetMetadata(buildMetadata(selectedWorkspace.ID, workspacePlan))
	return selectedWorkspace.ID, nil
}

func resolveWorkspace(client *Client, requestedWorkspace string) (Workspace, error) {
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return Workspace{}, err
	}

	if len(workspaces) == 0 {
		return Workspace{}, fmt.Errorf("no workspace available for this API key")
	}

	requestedWorkspace = strings.TrimSpace(requestedWorkspace)
	if requestedWorkspace == "" {
		return workspaces[0], nil
	}

	selectedWorkspace := slices.IndexFunc(workspaces, func(item Workspace) bool {
		return item.ID == requestedWorkspace
	})
	if selectedWorkspace < 0 {
		selectedWorkspace = slices.IndexFunc(workspaces, func(item Workspace) bool {
			return strings.EqualFold(item.Name, requestedWorkspace)
		})
	}

	if selectedWorkspace >= 0 {
		return workspaces[selectedWorkspace], nil
	}

	return Workspace{}, fmt.Errorf("workspace %q is not accessible with this API key", requestedWorkspace)
}

func (m Metadata) workspacePlan() string {
	if m.Workspace == nil {
		return workspacePlanProfessional
	}

	return strings.TrimSpace(m.Workspace.Plan)
}

func buildMetadata(workspaceID string, workspacePlan string) Metadata {
	resolvedPlan := strings.TrimSpace(workspacePlan)
	if resolvedPlan == "" {
		resolvedPlan = workspacePlanProfessional
	}

	return Metadata{
		Workspace: &WorkspaceMetadata{
			ID:   workspaceID,
			Plan: resolvedPlan,
		},
	}
}
