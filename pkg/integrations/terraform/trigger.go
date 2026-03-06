package terraform

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RunEvent struct{}

type RunEventConfiguration struct {
	WorkspaceID string   `mapstructure:"workspaceId"`
	Events      []string `mapstructure:"events"`
}

func (t *RunEvent) Name() string  { return "terraform.runEvent" }
func (t *RunEvent) Label() string { return "On Run Event" }
func (t *RunEvent) Description() string {
	return "Trigger a workflow when a Terraform Run transitions to selected states."
}
func (t *RunEvent) Icon() string  { return "terraform" }
func (t *RunEvent) Color() string { return "purple" }

func (t *RunEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "workspaceId",
			Label:    "Workspace",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "workspace"},
			},
		},
		{
			Name:     "events",
			Label:    "Run States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"run:created", "run:planning", "run:needs_attention", "run:applying", "run:completed", "run:errored"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Created",
							Value: "run:created",
						},
						{
							Label: "Planning",
							Value: "run:planning",
						},
						{
							Label: "Needs Attention",
							Value: "run:needs_attention",
						},
						{
							Label: "Applying",
							Value: "run:applying",
						},
						{
							Label: "Completed",
							Value: "run:completed",
						},
						{
							Label: "Errored",
							Value: "run:errored",
						},
					},
				},
			},
		},
	}
}

func (t *RunEvent) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *RunEvent) Cleanup(ctx core.TriggerContext) error  { return nil }

func (t *RunEvent) Setup(ctx core.TriggerContext) error {
	if err := ensureWorkspaceInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	); err != nil {
		return err
	}

	config := RunEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.WorkspaceID == "" {
		return fmt.Errorf("workspaceId is required")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
		Events:      config.Events,
	})
}

func (t *RunEvent) Actions() []core.Action { return []core.Action{} }
func (t *RunEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}
func (t *RunEvent) ExampleOutput() map[string]any { return nil }
func (t *RunEvent) ExampleData() map[string]any   { return nil }
func (t *RunEvent) Documentation() string         { return "" }
func (t *RunEvent) Triggers() []string            { return []string{} }

func (t *RunEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	data, code, err := ParseAndValidateWebhook(ctx)
	if err != nil {
		return code, err
	}
	if data == nil {
		return http.StatusOK, nil
	}

	config := RunEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, err
	}

	action, ok := data["action"].(string)
	if !ok {
		return http.StatusOK, nil
	}
	if len(config.Events) > 0 && !slices.Contains(config.Events, action) {
		return http.StatusOK, nil
	}

	workspaceID, _ := data["workspaceId"].(string)
	orgName, _ := data["organizationName"].(string)
	wsName, _ := data["workspaceName"].(string)

	matched := config.WorkspaceID == workspaceID || config.WorkspaceID == fmt.Sprintf("%s/%s", orgName, wsName)
	if !matched {
		return http.StatusOK, nil
	}

	runMessage, _ := data["runMessage"].(string)
	runID, _ := data["runId"].(string)
	runStatus, _ := data["runStatus"].(string)
	runURL, _ := data["runUrl"].(string)
	runCreatedBy, _ := data["runCreatedBy"].(string)

	emittedEvent := map[string]any{
		"runId":            runID,
		"workspaceId":      workspaceID,
		"action":           action,
		"runStatus":        runStatus,
		"runUrl":           runURL,
		"runMessage":       runMessage,
		"workspaceName":    wsName,
		"organizationName": orgName,
		"runCreatedBy":     runCreatedBy,
	}

	if runID != "" {
		client, err := getClientFromIntegration(ctx.Integration)
		if err == nil {
			run, err := client.ReadRun(runID)
			if err == nil && run != nil && run.Plan != nil && run.Plan.ID != "" {
				plan, err := client.ReadPlan(run.Plan.ID)
				if err == nil && plan != nil {
					emittedEvent["additions"] = plan.Attributes.ResourceAdditions
					emittedEvent["changes"] = plan.Attributes.ResourceChanges
					emittedEvent["destructions"] = plan.Attributes.ResourceDestructions
				}
			}
		}
	}

	if err := ctx.Events.Emit("terraform.runEvent", emittedEvent); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
