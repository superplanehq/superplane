package circleci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnPipelineCompleted struct{}

type OnPipelineCompletedMetadata struct {
	Project *Project `json:"project"`
}

type OnPipelineCompletedConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (p *OnPipelineCompleted) Name() string {
	return "circleci.onPipelineCompleted"
}

func (p *OnPipelineCompleted) Label() string {
	return "On Pipeline Completed"
}

func (p *OnPipelineCompleted) Description() string {
	return "Listen to CircleCI workflow completion events within a pipeline"
}

func (p *OnPipelineCompleted) Documentation() string {
	return `Triggers when a CircleCI workflow completes within a pipeline. Note that a single pipeline can contain multiple workflows.

## Terminology

- **Pipeline**: A CircleCI pipeline is triggered by a commit or API call and can contain multiple workflows
- **Workflow**: A set of jobs that run as part of a pipeline (defined in .circleci/config.yml)
- This trigger fires for each workflow completion, not when the entire pipeline finishes

## Use Cases

- **Workflow chaining**: Start SuperPlane workflows when CircleCI workflows complete
- **Status monitoring**: Monitor CI/CD workflow results
- **Notifications**: Send alerts when workflows succeed or fail
- **Post-processing**: Process artifacts after workflow completion

## Configuration

- **Project Slug**: The CircleCI project slug (e.g., gh/username/repo)

## Event Data

Each workflow completion event includes:
- **workflow**: Workflow information including ID, name, status, and URL
- **pipeline**: Parent pipeline information including ID, number, and trigger details
- **project**: Project information
- **organization**: Organization information

## Webhook Setup

This trigger automatically sets up a CircleCI webhook when configured. The webhook is managed by SuperPlane and cleaned up when the trigger is removed.`
}

func (p *OnPipelineCompleted) Icon() string {
	return "workflow"
}

func (p *OnPipelineCompleted) Color() string {
	return "gray"
}

func (p *OnPipelineCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g., gh/username/repo)",
			Placeholder: "gh/username/repo",
		},
	}
}

func (p *OnPipelineCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnPipelineCompletedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnPipelineCompletedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	// If this is the same project, nothing to do
	if metadata.Project != nil && config.ProjectSlug == metadata.Project.Slug {
		return nil
	}

	// Save project metadata
	err = ctx.Metadata.Set(OnPipelineCompletedMetadata{
		Project: &Project{
			Slug: config.ProjectSlug,
			Name: config.ProjectSlug,
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	// Request webhook setup
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: config.ProjectSlug,
	})
}

func (p *OnPipelineCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPipelineCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPipelineCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Verify webhook signature
	// CircleCI sends signature as "v1=<hex>" format
	signatureHeader := ctx.Headers.Get("circleci-signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Parse "v1=<hex>" format - extract just the hex part
	signature := signatureHeader
	if strings.HasPrefix(signatureHeader, "v1=") {
		signature = strings.TrimPrefix(signatureHeader, "v1=")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	// Parse webhook payload
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Verify this is a workflow-completed event
	eventType, ok := data["type"].(string)
	if !ok || eventType != "workflow-completed" {
		return http.StatusOK, nil // Ignore other event types
	}

	// Emit event to SuperPlane
	err = ctx.Events.Emit("circleci.workflow.completed", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (p *OnPipelineCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}
