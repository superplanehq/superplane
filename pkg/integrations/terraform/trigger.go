package terraform

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RunEvent struct{}

type RunEventConfiguration struct {
	WorkspaceID           string   `json:"workspaceId"`
	Events                []string `json:"events"`
	IncludeSuperPlaneRuns bool     `json:"includeSuperPlaneRuns"`
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
			Name:        "workspaceId",
			Label:       "Workspace ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the HCP Terraform workspace (e.g., ws-xyz123).",
		},
		{
			Name:     "events",
			Label:    "Run States",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"run:created", "run:completed", "run:errored"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Run Created", Value: "run:created"},
						{Label: "Run Planning", Value: "run:planning"},
						{Label: "Run Needs Attention", Value: "run:needs_attention"},
						{Label: "Run Applying", Value: "run:applying"},
						{Label: "Run Completed", Value: "run:completed"},
						{Label: "Run Errored", Value: "run:errored"},
						{Label: "Assessment Drifted", Value: "assessment:drifted"},
						{Label: "Assessment Failed", Value: "assessment:failed"},
					},
				},
			},
		},
		{
			Name:        "includeSuperPlaneRuns",
			Label:       "Include SuperPlane-initiated Runs",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "If enabled, also triggers for runs initiated by SuperPlane (e.g., via Queue Run). Warning: May cause infinite loops if not configured carefully.",
		},
	}
}

func (t *RunEvent) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *RunEvent) Cleanup(ctx core.TriggerContext) error  { return nil }

func (t *RunEvent) Setup(ctx core.TriggerContext) error {
	config := RunEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.WorkspaceID == "" {
		return fmt.Errorf("workspaceId is required")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
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

	config := RunEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, err
	}

	action, ok := data["action"].(string)
	if !ok || !slices.Contains(config.Events, action) {
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
	if !config.IncludeSuperPlaneRuns && slices.Contains([]rune(runMessage), '⚙') {
		return http.StatusOK, nil
	}

	runID, _ := data["runId"].(string)
	runStatus, _ := data["runStatus"].(string)
	runURL, _ := data["runUrl"].(string)
	runCreatedBy, _ := data["runCreatedBy"].(string)

	if err := ctx.Events.Emit("terraform.runEvent", map[string]any{
		"runId":            runID,
		"workspaceId":      workspaceID,
		"action":           action,
		"runStatus":        runStatus,
		"runUrl":           runURL,
		"runMessage":       runMessage,
		"workspaceName":    wsName,
		"organizationName": orgName,
		"runCreatedBy":     runCreatedBy,
	}); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

type NeedsAttention struct{}

type NeedsAttentionConfiguration struct {
	WorkspaceID string `json:"workspaceId"`
}

func (t *NeedsAttention) Name() string  { return "terraform.needsAttention" }
func (t *NeedsAttention) Label() string { return "On Run Needs Attention" }
func (t *NeedsAttention) Description() string {
	return "Trigger a workflow when a Terraform Run is paused for policy overrides or approval."
}
func (t *NeedsAttention) Icon() string  { return "terraform" }
func (t *NeedsAttention) Color() string { return "orange" }

func (t *NeedsAttention) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "workspaceId",
			Label:       "Workspace ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the HCP Terraform workspace.",
		},
	}
}

func (t *NeedsAttention) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *NeedsAttention) Cleanup(ctx core.TriggerContext) error  { return nil }

func (t *NeedsAttention) Setup(ctx core.TriggerContext) error {
	config := NeedsAttentionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	if config.WorkspaceID == "" {
		return fmt.Errorf("workspaceId is required")
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
	})
}

func (t *NeedsAttention) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "applyRun",
			Description:    "Approve and apply a run that is waiting for confirmation",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "runId",
					Label:       "Run ID",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The ID of the run to apply (e.g., run-xxxxx)",
				},
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Description: "Optional comment to add when applying the run",
				},
			},
		},
		{
			Name:           "discardRun",
			Description:    "Discard a run that is waiting for confirmation",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "runId",
					Label:       "Run ID",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The ID of the run to discard (e.g., run-xxxxx)",
				},
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Description: "Optional comment to add when discarding the run",
				},
			},
		},
		{
			Name:           "overridePolicy",
			Description:    "Override a soft-mandatory Sentinel policy check",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "policyCheckId",
					Label:       "Policy Check ID",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The ID of the policy check to override (e.g., polchk-xxxxx)",
				},
			},
		},
	}
}

func (t *NeedsAttention) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch ctx.Name {
	case "applyRun":
		runID, _ := ctx.Parameters["runId"].(string)
		comment, _ := ctx.Parameters["comment"].(string)
		if runID == "" {
			return nil, fmt.Errorf("runId is required")
		}

		err := client.ApplyRun(context.Background(), runID, comment)
		if err != nil {
			return nil, fmt.Errorf("failed to apply run: %w", err)
		}
		return map[string]any{"status": "applied", "runId": runID}, nil

	case "discardRun":
		runID, _ := ctx.Parameters["runId"].(string)
		comment, _ := ctx.Parameters["comment"].(string)
		if runID == "" {
			return nil, fmt.Errorf("runId is required")
		}

		err := client.DiscardRun(context.Background(), runID, comment)
		if err != nil {
			return nil, fmt.Errorf("failed to discard run: %w", err)
		}
		return map[string]any{"status": "discarded", "runId": runID}, nil

	case "overridePolicy":
		policyCheckID, _ := ctx.Parameters["policyCheckId"].(string)
		if policyCheckID == "" {
			return nil, fmt.Errorf("policyCheckId is required")
		}

		err := client.OverridePolicy(context.Background(), policyCheckID)
		if err != nil {
			return nil, fmt.Errorf("failed to override policy: %w", err)
		}
		return map[string]any{"status": "overridden", "policyCheckId": policyCheckID}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
func (t *NeedsAttention) ExampleOutput() map[string]any { return nil }
func (t *NeedsAttention) ExampleData() map[string]any   { return nil }
func (t *NeedsAttention) Documentation() string         { return "" }
func (t *NeedsAttention) Triggers() []string            { return []string{} }

func (t *NeedsAttention) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	data, code, err := ParseAndValidateWebhook(ctx)
	if err != nil {
		return code, err
	}

	config := NeedsAttentionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, err
	}

	action, ok := data["action"].(string)
	if !ok || action != "run:needs_attention" {
		return http.StatusOK, nil
	}

	workspaceID, _ := data["workspaceId"].(string)
	orgName, _ := data["organizationName"].(string)
	wsName, _ := data["workspaceName"].(string)

	matched := config.WorkspaceID == workspaceID || config.WorkspaceID == fmt.Sprintf("%s/%s", orgName, wsName)
	if !matched {
		return http.StatusOK, nil
	}

	runStatus, _ := data["runStatus"].(string)
	runURL, _ := data["runUrl"].(string)
	runCreatedBy, _ := data["runCreatedBy"].(string)
	runMessage, _ := data["runMessage"].(string)
	runID, _ := data["runId"].(string)

	if err := ctx.Events.Emit("terraform.needsAttention", map[string]any{
		"runId":            runID,
		"workspaceId":      workspaceID,
		"action":           action,
		"runStatus":        runStatus,
		"runUrl":           runURL,
		"runMessage":       runMessage,
		"workspaceName":    wsName,
		"organizationName": orgName,
		"runCreatedBy":     runCreatedBy,
	}); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
