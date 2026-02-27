package terraform

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type TerraformRunEvent struct{}

type TerraformRunEventConfiguration struct {
	WorkspaceID           string   `json:"workspaceId"`
	Events                []string `json:"events"`
	IncludeSuperPlaneRuns bool     `json:"includeSuperPlaneRuns"`
}

func (t *TerraformRunEvent) Name() string  { return "terraform.runEvent" }
func (t *TerraformRunEvent) Label() string { return "On Run Event" }
func (t *TerraformRunEvent) Description() string {
	return "Trigger a workflow when a Terraform Run transitions to selected states."
}
func (t *TerraformRunEvent) Icon() string  { return "terraform" }
func (t *TerraformRunEvent) Color() string { return "purple" }

func (t *TerraformRunEvent) Configuration() []configuration.Field {
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

func (t *TerraformRunEvent) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *TerraformRunEvent) Cleanup(ctx core.TriggerContext) error  { return nil }

func (t *TerraformRunEvent) Setup(ctx core.TriggerContext) error {
	config := TerraformRunEventConfiguration{}
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

func (t *TerraformRunEvent) Actions() []core.Action { return []core.Action{} }
func (t *TerraformRunEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}
func (t *TerraformRunEvent) ExampleOutput() map[string]any { return nil }
func (t *TerraformRunEvent) ExampleData() map[string]any   { return nil }
func (t *TerraformRunEvent) Documentation() string         { return "" }
func (t *TerraformRunEvent) Triggers() []string            { return []string{} }

func (t *TerraformRunEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	data, code, err := ParseAndValidateWebhook(ctx)
	if err != nil {
		return code, err
	}

	config := TerraformRunEventConfiguration{}
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

type TerraformNeedsAttention struct{}

type TerraformNeedsAttentionConfiguration struct {
	WorkspaceID string `json:"workspaceId"`
}

func (t *TerraformNeedsAttention) Name() string  { return "terraform.needsAttention" }
func (t *TerraformNeedsAttention) Label() string { return "On Run Needs Attention" }
func (t *TerraformNeedsAttention) Description() string {
	return "Trigger a workflow when a Terraform Run is paused for policy overrides or approval."
}
func (t *TerraformNeedsAttention) Icon() string  { return "terraform" }
func (t *TerraformNeedsAttention) Color() string { return "orange" }

func (t *TerraformNeedsAttention) Configuration() []configuration.Field {
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

func (t *TerraformNeedsAttention) Cancel(ctx core.ExecutionContext) error { return nil }
func (t *TerraformNeedsAttention) Cleanup(ctx core.TriggerContext) error  { return nil }

func (t *TerraformNeedsAttention) Setup(ctx core.TriggerContext) error {
	config := TerraformNeedsAttentionConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
	})
}

func (t *TerraformNeedsAttention) Actions() []core.Action {
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

func (t *TerraformNeedsAttention) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	switch ctx.Name {
	case "applyRun":
		runID, _ := ctx.Parameters["runId"].(string)
		comment, _ := ctx.Parameters["comment"].(string)
		if runID == "" {
			return nil, fmt.Errorf("runId is required")
		}

		err := client.TFE.Runs.Apply(context.Background(), runID, tfe.RunApplyOptions{
			Comment: &comment,
		})
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

		err := client.TFE.Runs.Discard(context.Background(), runID, tfe.RunDiscardOptions{
			Comment: &comment,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to discard run: %w", err)
		}
		return map[string]any{"status": "discarded", "runId": runID}, nil

	case "overridePolicy":
		policyCheckID, _ := ctx.Parameters["policyCheckId"].(string)
		if policyCheckID == "" {
			return nil, fmt.Errorf("policyCheckId is required")
		}

		_, err := client.TFE.PolicyChecks.Override(context.Background(), policyCheckID)
		if err != nil {
			return nil, fmt.Errorf("failed to override policy: %w", err)
		}
		return map[string]any{"status": "overridden", "policyCheckId": policyCheckID}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
func (t *TerraformNeedsAttention) ExampleOutput() map[string]any { return nil }
func (t *TerraformNeedsAttention) ExampleData() map[string]any   { return nil }
func (t *TerraformNeedsAttention) Documentation() string         { return "" }
func (t *TerraformNeedsAttention) Triggers() []string            { return []string{} }

func (t *TerraformNeedsAttention) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	data, code, err := ParseAndValidateWebhook(ctx)
	if err != nil {
		return code, err
	}

	config := TerraformNeedsAttentionConfiguration{}
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

	if err := ctx.Events.Emit("terraform.needsAttention", map[string]any{
		"runId":            data["runId"],
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
