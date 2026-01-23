package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnWorkflowRun struct{}

type OnWorkflowRunConfiguration struct {
	Repository    string   `json:"repository" mapstructure:"repository"`
	Conclusions   []string `json:"conclusions" mapstructure:"conclusions"`
	WorkflowFiles []string `json:"workflowFiles" mapstructure:"workflowFiles"`
}

func (w *OnWorkflowRun) Name() string {
	return "github.onWorkflowRun"
}

func (w *OnWorkflowRun) Label() string {
	return "On Workflow Run"
}

func (w *OnWorkflowRun) Description() string {
	return "Listen to workflow run events"
}

func (w *OnWorkflowRun) Icon() string {
	return "github"
}

func (w *OnWorkflowRun) Color() string {
	return "gray"
}

func (w *OnWorkflowRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:    "conclusions",
			Label:   "Conclusions",
			Type:    configuration.FieldTypeMultiSelect,
			Default: []string{"success", "failure"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Success", Value: "success"},
						{Label: "Failure", Value: "failure"},
						{Label: "Cancelled", Value: "cancelled"},
						{Label: "Skipped", Value: "skipped"},
						{Label: "Timed Out", Value: "timed_out"},
						{Label: "Action Required", Value: "action_required"},
						{Label: "Stale", Value: "stale"},
						{Label: "Neutral", Value: "neutral"},
						{Label: "Startup Failure", Value: "startup_failure"},
					},
				},
			},
		},
		{
			Name:        "workflowFiles",
			Label:       "Workflow Files",
			Description: "Path to workflow files, e.g. .github/workflows/ci.yml",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     []string{".github/workflows/ci.yml"},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Workflow file",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (w *OnWorkflowRun) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnWorkflowRunConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.AppInstallation.RequestWebhook(WebhookConfiguration{
		EventType:  "workflow_run",
		Repository: config.Repository,
	})
}

func (w *OnWorkflowRun) Actions() []core.Action {
	return []core.Action{}
}

func (w *OnWorkflowRun) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (w *OnWorkflowRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnWorkflowRunConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "workflow_run" {
		return http.StatusOK, nil
	}

	code, err := verifySignature(ctx)
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Only emit events for completed workflow runs
	action, ok := data["action"].(string)
	if !ok || action != "completed" {
		return http.StatusOK, nil
	}

	// Filter by conclusion if specified
	if len(config.Conclusions) > 0 {
		if !matchesConclusion(data, config.Conclusions) {
			return http.StatusOK, nil
		}
	}

	// Filter by workflow file if specified
	if len(config.WorkflowFiles) > 0 {
		if !matchesWorkflowFile(data, config.WorkflowFiles) {
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit("github.workflowRun", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func matchesConclusion(data map[string]any, allowedConclusions []string) bool {
	workflowRun, ok := data["workflow_run"].(map[string]any)
	if !ok {
		return false
	}

	conclusion, ok := workflowRun["conclusion"].(string)
	if !ok {
		return false
	}

	return slices.Contains(allowedConclusions, conclusion)
}

func matchesWorkflowFile(data map[string]any, allowedWorkflowFiles []string) bool {
	workflowRun, ok := data["workflow_run"].(map[string]any)
	if !ok {
		return false
	}

	path, ok := workflowRun["path"].(string)
	if !ok {
		return false
	}

	return slices.Contains(allowedWorkflowFiles, path)
}
